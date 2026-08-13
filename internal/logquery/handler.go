package logquery

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/response"
	wsutil "ops-platform/internal/pkg/websocket"
)

type Handler struct {
	service  *Service
	upgrader websocket.Upgrader
}

type streamEvent struct {
	line *Line
	err  error
}

func NewHandler(service *Service, allowedOrigins []string) *Handler {
	upgrader := wsutil.NewUpgrader(allowedOrigins)
	return &Handler{service: service, upgrader: upgrader}
}

func (h *Handler) Register(r gin.IRoutes, readMiddleware ...gin.HandlerFunc) {
	mw := append([]gin.HandlerFunc{}, readMiddleware...)
	r.GET("/namespaces/:namespace/pods/:pod/logs", append(mw, h.podLogs)...)
	r.GET("/logs", append(mw, h.logs)...)
}

func (h *Handler) RegisterWebSocket(r gin.IRoutes, readMiddleware ...gin.HandlerFunc) {
	mw := append([]gin.HandlerFunc{}, readMiddleware...)
	r.GET("/namespaces/:namespace/pods/:pod/logs/follow", append(mw, h.followPodLogs)...)
}

func (h *Handler) podLogs(c *gin.Context) {
	query := h.queryFromContext(c)
	query.Namespace = c.Param("namespace")
	query.Pod = c.Param("pod")
	h.runQuery(c, query)
}

func (h *Handler) logs(c *gin.Context) {
	query := h.queryFromContext(c)
	query.Namespace = c.Query("namespace")
	query.Pod = c.Query("pod")
	h.runQuery(c, query)
}

func (h *Handler) runQuery(c *gin.Context, query Query) {
	result, err := h.service.Query(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) followPodLogs(c *gin.Context) {
	query := h.queryFromContext(c)
	query.Namespace = c.Param("namespace")
	query.Pod = c.Param("pod")
	query.Previous = false

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, http.Header{
		"Sec-WebSocket-Protocol": []string{"ops-platform.logs.v1"},
	})
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	events := make(chan streamEvent, 64)
	go func() {
		err := h.service.Follow(ctx, query, func(line Line) error {
			select {
			case events <- streamEvent{line: &line}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		select {
		case events <- streamEvent{err: err}:
		case <-ctx.Done():
		}
	}()
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	if err := writeStreamMessage(conn, StreamMessage{Type: "ready"}); err != nil {
		return
	}
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case event := <-events:
			if event.line != nil {
				if err := writeStreamMessage(conn, StreamMessage{Type: "line", Line: event.line}); err != nil {
					cancel()
					return
				}
				continue
			}
			if event.err != nil && !errors.Is(event.err, context.Canceled) {
				appErr := apperrors.From(event.err)
				_ = writeStreamMessage(conn, StreamMessage{Type: "error", Code: int(appErr.Code), Message: appErr.Message})
				return
			}
			_ = writeStreamMessage(conn, StreamMessage{Type: "complete"})
			return
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func writeStreamMessage(conn *websocket.Conn, message StreamMessage) error {
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteJSON(message)
}

func (h *Handler) queryFromContext(c *gin.Context) Query {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "200"), 10, 64)
	if err != nil {
		limit = 200
	}
	previous := c.Query("previous") == "true"
	return Query{
		Container: c.Query("container"),
		From:      c.Query("from"),
		Keyword:   c.Query("keyword"),
		Level:     c.Query("level"),
		Limit:     limit,
		Previous:  previous,
	}
}

func invalidLogQuery(message string) error {
	return apperrors.New(apperrors.CodeInvalidArgument, message, http.StatusBadRequest)
}
