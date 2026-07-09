package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kumargaurav/summit-backend/internal/config"
)

// NewSender picks the SMS implementation from config: MSG91 when configured,
// otherwise the dev LogSender (codes to stdout only).
func NewSender(cfg config.Config) SMSSender {
	if cfg.SMSProvider == "msg91" {
		return MSG91Sender{AuthKey: cfg.MSG91AuthKey, TemplateID: cfg.MSG91TemplateID}
	}
	return LogSender{}
}

// MSG91Sender delivers OTPs via MSG91's OTP API (docs.msg91.com). The DLT
// template referenced by TemplateID must contain the ##OTP## variable.
type MSG91Sender struct {
	AuthKey    string
	TemplateID string
	Client     *http.Client
}

func (m MSG91Sender) Send(ctx context.Context, phone, code string) error {
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	q := url.Values{
		"template_id": {m.TemplateID},
		"mobile":      {strings.TrimPrefix(phone, "+")},
		"otp":         {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://control.msg91.com/api/v5/otp?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("authkey", m.AuthKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("msg91 request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("msg91 status %d: %s", resp.StatusCode, body)
	}
	var out struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err == nil && out.Type == "error" {
		return fmt.Errorf("msg91 error: %s", out.Message)
	}
	return nil
}
