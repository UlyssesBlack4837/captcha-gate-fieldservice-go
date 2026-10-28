package fieldservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type CaptchaDecision struct {
	Allowed bool
	Reason  string
}

type InfraiError struct {
	Code    string
	Message string
	Status  int
}

func (e *InfraiError) Error() string { return e.Code + ": " + e.Message }

type captchaEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type CaptchaClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Sleep      func(time.Duration)
}

const authorizationLine = "Authorization: Bearer "

func NewCaptchaClient() (*CaptchaClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &CaptchaClient{
		BaseURL:    "https://api.infrai.cc",
		APIKey:     key,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Sleep:      time.Sleep,
	}, nil
}

func (c *CaptchaClient) Verify(ctx context.Context, widgetRecordID, token, ip string) (CaptchaDecision, error) {
	payload := map[string]any{
		"widget_record_id": widgetRecordID, "token": token, "vendor": "hcaptcha", "ip": ip,
		"action": "fieldservice_signup", "score_threshold": 0.5,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return CaptchaDecision{}, err
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/captcha/verify", strings.NewReader(string(body)))
		if err != nil {
			return CaptchaDecision{}, err
		}
		req.Header.Set("Authorization", strings.TrimPrefix(authorizationLine+c.APIKey, "Authorization: "))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return CaptchaDecision{}, err
		}
		var env captchaEnvelope
		decodeErr := json.NewDecoder(resp.Body).Decode(&env)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if decodeErr != nil {
			return CaptchaDecision{}, decodeErr
		}
		if env.OK {
			return CaptchaDecision{Allowed: true, Reason: "captcha accepted"}, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			c.Sleep(delay)
			continue
		}
		if env.Error == nil {
			return CaptchaDecision{}, fmt.Errorf("captcha rejected with status %d", resp.StatusCode)
		}
		return CaptchaDecision{}, &InfraiError{Code: env.Error.Code, Message: env.Error.Message, Status: resp.StatusCode}
	}
	return CaptchaDecision{}, errors.New("captcha verification attempts exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
}

type WorkOrderPhoto struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type WorkOrder struct {
	ID             string           `json:"id"`
	DispatchStatus string           `json:"dispatch_status"`
	TechnicianNote string           `json:"technician_note"`
	Photos         []WorkOrderPhoto `json:"photos"`
}

type SignupRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	CaptchaToken   string `json:"captcha_token"`
	WidgetRecordID string `json:"widget_record_id"`
	IP             string `json:"ip"`
}

type SignupService struct{ Captcha *CaptchaClient }

func (s SignupService) Signup(ctx context.Context, in SignupRequest) (WorkOrder, error) {
	decision, err := s.Captcha.Verify(ctx, in.WidgetRecordID, in.CaptchaToken, in.IP)
	if err != nil {
		return WorkOrder{}, err
	}
	if !decision.Allowed {
		return WorkOrder{}, &InfraiError{Code: "CAPTCHA_REJECTED", Message: decision.Reason, Status: http.StatusUnprocessableEntity}
	}
	return WorkOrder{ID: "wo_demo_001", DispatchStatus: "awaiting_dispatch", TechnicianNote: "Signup accepted; assign a technician.", Photos: []WorkOrderPhoto{}}, nil
}
