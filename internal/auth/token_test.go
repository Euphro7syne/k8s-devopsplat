package auth

import "testing"

func TestWebSocketBearerToken(t *testing.T) {
	token, err := WebSocketBearerToken("ops-platform.logs.v1, bearer.header.payload.signature")
	if err != nil {
		t.Fatalf("parse websocket bearer token: %v", err)
	}
	if token != "header.payload.signature" {
		t.Fatalf("unexpected token: %q", token)
	}
	if _, err := WebSocketBearerToken("ops-platform.logs.v1"); err == nil {
		t.Fatal("expected missing websocket bearer protocol to be rejected")
	}
}
