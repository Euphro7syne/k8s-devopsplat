package logquery

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/sanitizer"
	"ops-platform/internal/resources"
)

const (
	defaultLogLimit int64 = 200
	maxLogLimit     int64 = 2000
)

type Service struct {
	client resources.KubernetesClient
}

func NewService(client resources.KubernetesClient) *Service {
	return &Service{client: client}
}

func (s *Service) Query(ctx context.Context, query Query) (Result, error) {
	if err := s.validate(query); err != nil {
		return Result{}, err
	}
	limit := normalizeLimit(query.Limit)
	options := &corev1.PodLogOptions{
		Container:  query.Container,
		Previous:   query.Previous,
		Timestamps: true,
		TailLines:  &limit,
	}
	if query.From != "" {
		parsed, err := time.Parse(time.RFC3339, query.From)
		if err != nil {
			return Result{}, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "from must be RFC3339", http.StatusBadRequest)
		}
		options.SinceTime = &metav1.Time{Time: parsed}
	}

	stream, err := s.client.CoreV1().Pods(query.Namespace).GetLogs(query.Pod, options).Stream(ctx)
	if err != nil {
		return Result{}, mapKubernetesError(err, "query pod logs failed")
	}
	defer stream.Close()

	lines, err := readLines(stream, query.Keyword, query.Level)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Source:    "kubernetes",
		Namespace: query.Namespace,
		Pod:       query.Pod,
		Container: query.Container,
		Lines:     lines,
		Total:     len(lines),
	}, nil
}

func (s *Service) Follow(ctx context.Context, query Query, onLine func(Line) error) error {
	if err := s.validate(query); err != nil {
		return err
	}
	if query.Previous {
		return apperrors.New(apperrors.CodeInvalidArgument, "previous container logs do not support realtime follow", http.StatusBadRequest)
	}
	if onLine == nil {
		return apperrors.New(apperrors.CodeInvalidArgument, "stream line handler is required", http.StatusBadRequest)
	}

	limit := normalizeLimit(query.Limit)
	options := &corev1.PodLogOptions{
		Container:  query.Container,
		Follow:     true,
		Timestamps: true,
		TailLines:  &limit,
	}
	if query.From != "" {
		parsed, err := time.Parse(time.RFC3339, query.From)
		if err != nil {
			return apperrors.Wrap(err, apperrors.CodeInvalidArgument, "from must be RFC3339", http.StatusBadRequest)
		}
		options.SinceTime = &metav1.Time{Time: parsed}
	}

	stream, err := s.client.CoreV1().Pods(query.Namespace).GetLogs(query.Pod, options).Stream(ctx)
	if err != nil {
		return mapKubernetesError(err, "follow pod logs failed")
	}
	defer stream.Close()
	return streamLines(ctx, stream, query.Keyword, query.Level, onLine)
}

func streamLines(ctx context.Context, reader io.Reader, keyword, level string, onLine func(Line) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		if !lineMatches(raw, keyword, level) {
			continue
		}
		if err := onLine(Line{Raw: sanitizer.String(raw)}); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("read realtime log stream: %w", err)
	}
	return nil
}

func (s *Service) validate(query Query) error {
	if s.client == nil {
		return apperrors.New(apperrors.CodeKubernetesUnavailable, "kubernetes client unavailable", http.StatusServiceUnavailable)
	}
	if query.Namespace == "" || query.Pod == "" {
		return apperrors.New(apperrors.CodeInvalidArgument, "namespace and pod are required", http.StatusBadRequest)
	}
	return nil
}

func normalizeLimit(limit int64) int64 {
	if limit <= 0 {
		return defaultLogLimit
	}
	if limit > maxLogLimit {
		return maxLogLimit
	}
	return limit
}

func readLines(reader io.Reader, keyword, level string) ([]Line, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	lines := make([]Line, 0)
	for scanner.Scan() {
		raw := scanner.Text()
		if !lineMatches(raw, keyword, level) {
			continue
		}
		lines = append(lines, Line{Raw: sanitizer.String(raw)})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read log stream: %w", err)
	}
	return lines, nil
}

func lineMatches(line, keyword, level string) bool {
	lower := strings.ToLower(line)
	if keyword != "" && !strings.Contains(lower, strings.ToLower(keyword)) {
		return false
	}
	if level == "" {
		return true
	}
	level = strings.ToLower(level)
	return strings.Contains(lower, level) ||
		strings.Contains(lower, "level="+level) ||
		strings.Contains(lower, `"level":"`+level+`"`) ||
		strings.Contains(lower, "["+level+"]")
}

func mapKubernetesError(err error, message string) error {
	if err == nil {
		return nil
	}
	switch {
	case apierrors.IsNotFound(err):
		return apperrors.Wrap(err, apperrors.CodeNotFound, message, http.StatusNotFound)
	case apierrors.IsForbidden(err):
		return apperrors.Wrap(err, apperrors.CodePermissionDenied, "kubernetes permission denied", http.StatusForbidden)
	default:
		return apperrors.Wrap(err, apperrors.CodeLogQueryFailed, message, http.StatusServiceUnavailable)
	}
}
