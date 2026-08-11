package websocket

import (
	"net/http"
	"net/url"
)

import gorilla "github.com/gorilla/websocket"

func NewUpgrader(allowedOrigins []string) gorilla.Upgrader {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAny := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAny = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return gorilla.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if allowAny {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if _, ok := allowed[origin]; ok {
				return true
			}
			parsed, err := url.Parse(origin)
			if err != nil {
				return false
			}
			hostOrigin := parsed.Scheme + "://" + parsed.Host
			_, ok := allowed[hostOrigin]
			return ok
		},
	}
}
