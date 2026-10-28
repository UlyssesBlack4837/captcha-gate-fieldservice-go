package fieldservice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignupRejectsCaptchaBeforeWorkOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/captcha/verify" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "data": nil, "error": map[string]string{"code": "CAPTCHA_REJECTED", "message": "captcha rejected"}, "metadata": nil})
	}))
	defer server.Close()
	client := &CaptchaClient{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client(), Sleep: func(_ time.Duration) {}}
	_, err := (SignupService{Captcha: client}).Signup(context.Background(), SignupRequest{Email: "tech@example.com", Password: "secret", Name: "Mina", CaptchaToken: "bot-token", WidgetRecordID: "widget-record", IP: "203.0.113.9"})
	var infraiErr *InfraiError
	if !errors.As(err, &infraiErr) || infraiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("err = %#v", err)
	}
}
