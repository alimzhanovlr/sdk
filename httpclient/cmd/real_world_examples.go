package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alimzhanovlr/sdk/httpclient"
)

// ====================================================================================
// ПРИМЕР 1: API Клиент для платежной системы (Stripe-like)
// ====================================================================================

type PaymentClient struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

func NewPaymentClient(apiKey string) *PaymentClient {
	logger := httpclient.NewSimpleLogger(httpclient.INFO)

	// Специфичная конфигурация для платежного API
	config := &httpclient.LoggingConfig{
		Logger:          logger,
		LogRequestBody:  true,
		LogResponseBody: true,
		LogHeaders:      true,

		SanitizerConfig: &httpclient.SanitizerConfig{
			SensitiveFields: []string{
				// Стандартные поля
				"password", "token", "secret", "api_key",

				// Специфичные для платежей
				"card_number", "cvv", "cvc", "card_cvv",
				"iban", "account_number", "routing_number",
				"stripe_key", "publishable_key", "secret_key",
				"client_secret", "payment_method_token",
			},
			SensitivePatterns: []*regexp.Regexp{
				// Stripe keys
				regexp.MustCompile(`(sk_live_)[a-zA-Z0-9]{24,}`),
				regexp.MustCompile(`(pk_live_)[a-zA-Z0-9]{24,}`),

				// Credit card numbers
				regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
			},
			Mask:           "[REDACTED]",
			MaxBodySize:    50 * 1024,
			HeaderMaskMode: httpclient.HeaderMaskPartial,
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &PaymentClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: "https://api.stripe.com/v1",
	}
}

func (p *PaymentClient) CreateCharge(ctx context.Context, amount int, currency, cardToken string) error {
	formData := fmt.Sprintf("amount=%d&currency=%s&source=%s", amount, currency, cardToken)

	req, _ := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/charges", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ====================================================================================
// ПРИМЕР 2: Микросервисный клиент с трассировкой
// ====================================================================================

type MicroserviceClient struct {
	client  *http.Client
	service string
}

func NewMicroserviceClient(serviceName string, env string) *MicroserviceClient {
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	config := httpclient.DefaultLoggingConfig(logger)

	// В проде логируем меньше
	if env == "production" {
		config.LogRequestBody = false
		config.LogResponseBody = false
		config.Verbose = false
		logger = httpclient.NewSimpleLogger(httpclient.ERROR)
	}

	// Не логируем health checks
	config.ShouldLog = func(req *http.Request) bool {
		return !strings.HasSuffix(req.URL.Path, "/health")
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &MicroserviceClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   10 * time.Second,
		},
		service: serviceName,
	}
}

func (m *MicroserviceClient) CallService(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", m.service+endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем trace ID для корреляции логов
	if traceID := ctx.Value("trace_id"); traceID != nil {
		req.Header.Set("X-Trace-ID", fmt.Sprint(traceID))
	}

	return m.client.Do(req)
}

// ====================================================================================
// ПРИМЕР 3: API клиент с файловыми загрузками
// ====================================================================================

type FileUploadClient struct {
	client *http.Client
}

func NewFileUploadClient() *FileUploadClient {
	logger := httpclient.NewSimpleLogger(httpclient.INFO)

	config := &httpclient.LoggingConfig{
		Logger:         logger,
		LogHeaders:     true,
		LogRequestBody: true,

		// Умная обработка файлов
		ShouldLogBody: func(req *http.Request, contentType string, size int) bool {
			// Не логируем бинарные файлы
			if strings.HasPrefix(contentType, "image/") ||
				strings.HasPrefix(contentType, "video/") ||
				strings.HasPrefix(contentType, "application/pdf") {
				return false
			}

			// Не логируем очень большие тела
			if size > 1*1024*1024 { // 1MB
				return false
			}

			// Не логируем multipart с файлами
			if strings.Contains(contentType, "multipart/form-data") && size > 10*1024 {
				return false
			}

			return true
		},

		SanitizerConfig: httpclient.DefaultSanitizerConfig(),
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &FileUploadClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   5 * time.Minute, // Большой таймаут для файлов
		},
	}
}

func (f *FileUploadClient) UploadFile(ctx context.Context, url string, fileData []byte, contentType string) error {
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(fileData))
	req.Header.Set("Content-Type", contentType)

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ====================================================================================
// ПРИМЕР 4: OAuth клиент
// ====================================================================================

type OAuthClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
}

func NewOAuthClient(clientID, clientSecret string) *OAuthClient {
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	config := &httpclient.LoggingConfig{
		Logger:         logger,
		LogRequestBody: true,
		LogHeaders:     true,

		SanitizerConfig: &httpclient.SanitizerConfig{
			SensitiveFields: []string{
				"client_secret", "client_id", "access_token",
				"refresh_token", "code", "grant_type",
				"password", "username",
			},
			SensitivePatterns: []*regexp.Regexp{
				// OAuth Bearer tokens
				regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`),
				// Authorization codes
				regexp.MustCompile(`(code=)[a-zA-Z0-9\-._~+/]+=*`),
			},
			Mask:        "***HIDDEN***",
			MaxBodySize: 10 * 1024,
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &OAuthClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   30 * time.Second,
		},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (o *OAuthClient) GetAccessToken(ctx context.Context, authCode string) (string, error) {
	formData := fmt.Sprintf(
		"grant_type=authorization_code&code=%s&client_id=%s&client_secret=%s",
		authCode, o.clientID, o.clientSecret,
	)

	req, _ := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://oauth.example.com/token",
		strings.NewReader(formData),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.AccessToken, nil
}

// ====================================================================================
// ПРИМЕР 5: XML SOAP клиент
// ====================================================================================

type SOAPClient struct {
	client *http.Client
}

func NewSOAPClient() *SOAPClient {
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	config := &httpclient.LoggingConfig{
		Logger:          logger,
		LogRequestBody:  true,
		LogResponseBody: true,

		SanitizerConfig: &httpclient.SanitizerConfig{
			SensitiveFields: []string{
				"Password", "ApiKey", "Token", "Secret",
				"AuthToken", "SessionId", "Credentials",
			},
			Mask:        "[REDACTED]",
			MaxBodySize: 100 * 1024,
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &SOAPClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   60 * time.Second,
		},
	}
}

func (s *SOAPClient) CallSOAPService(ctx context.Context, endpoint string, username, password string) error {
	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
    <soap:Header>
        <Authentication>
            <Username>%s</Username>
            <Password>%s</Password>
        </Authentication>
    </soap:Header>
    <soap:Body>
        <GetData>
            <RequestId>12345</RequestId>
        </GetData>
    </soap:Body>
</soap:Envelope>`, username, password)

	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(soapEnvelope))
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "GetData")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ====================================================================================
// ПРИМЕР 6: Webhook клиент с подписями
// ====================================================================================

type WebhookClient struct {
	client        *http.Client
	signingSecret string
}

func NewWebhookClient(signingSecret string) *WebhookClient {
	logger := httpclient.NewSimpleLogger(httpclient.INFO)

	config := &httpclient.LoggingConfig{
		Logger:         logger,
		LogRequestBody: true,
		LogHeaders:     true,

		SanitizerConfig: &httpclient.SanitizerConfig{
			SensitiveFields: []string{
				"signing_secret", "webhook_secret", "secret",
				"signature", "hmac",
			},
			SensitivePatterns: []*regexp.Regexp{
				// Webhook signatures
				regexp.MustCompile(`(?i)(x-signature:\s*)[a-f0-9]{64,}`),
				regexp.MustCompile(`(?i)(whsec_)[a-zA-Z0-9]{32,}`),
			},
			Mask: "[REDACTED]",
			SensitiveHeaders: []string{
				"x-signature", "x-webhook-signature",
				"stripe-signature",
			},
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)

	return &WebhookClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   10 * time.Second,
		},
		signingSecret: signingSecret,
	}
}

func (w *WebhookClient) SendWebhook(ctx context.Context, url string, payload interface{}) error {
	body, _ := json.Marshal(payload)

	// Генерируем подпись (упрощенно)
	signature := "sha256=abc123def456..." // В реальности - HMAC

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ====================================================================================
// MAIN - Демонстрация всех примеров
// ====================================================================================

func main() {
	ctx := context.Background()

	fmt.Println("=== Payment Client Example ===")
	paymentClient := NewPaymentClient("sk_live_test_key_12345")
	paymentClient.CreateCharge(ctx, 1000, "usd", "tok_visa_12345")

	fmt.Println("\n=== Microservice Client Example ===")
	msClient := NewMicroserviceClient("http://user-service:8080", "development")
	msClient.CallService(ctx, "/users", map[string]string{
		"email":    "user@example.com",
		"password": "secret123",
	})

	fmt.Println("\n=== File Upload Client Example ===")
	fileClient := NewFileUploadClient()
	fileClient.UploadFile(ctx, "https://api.example.com/upload", []byte("file content"), "text/plain")

	fmt.Println("\n=== OAuth Client Example ===")
	oauthClient := NewOAuthClient("client_id_123", "client_secret_xyz")
	oauthClient.GetAccessToken(ctx, "auth_code_abc")

	fmt.Println("\n=== SOAP Client Example ===")
	soapClient := NewSOAPClient()
	soapClient.CallSOAPService(ctx, "https://soap.example.com/service", "admin", "password123")

	fmt.Println("\n=== Webhook Client Example ===")
	webhookClient := NewWebhookClient("whsec_secret_key_12345")
	webhookClient.SendWebhook(ctx, "https://customer.com/webhook", map[string]interface{}{
		"event": "user.created",
		"data": map[string]string{
			"user_id": "123",
			"email":   "user@example.com",
		},
	})
}
