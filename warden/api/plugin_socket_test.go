package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gin-gonic/gin"
)

// Regression test: the upgrade used to run inside a gin handler, where
// gin 1.11 rejected the hijack and every connection dropped after the 101.
func TestPluginSocketUpgradesThroughGinRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.PluginToken = "test-token"

	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		typ, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		_ = conn.Write(r.Context(), typ, data)
	})
	server := httptest.NewServer(withPluginSocket(gin.New(), echo))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + pluginSocketPath

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer test-token"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, data, err := conn.Read(ctx); err != nil || string(data) != `{"type":"ping"}` {
		t.Fatalf("echo: %q, %v", data, err)
	}

	_, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer wrong"}},
	})
	if err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token: want 401, got %v (%v)", resp, err)
	}
}
