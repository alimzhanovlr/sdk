package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/alimzhanovlr/sdk/httpclient"
)

// APIClient пример клиента для внешнего API
type APIClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewAPIClient создает новый API клиент с логированием
func NewAPIClient(baseURL, apiKey string, logger httpclient.Logger) *APIClient {
	// Кастомная конфигурация для этого API
	sanitizerConfig := &httpclient.SanitizerConfig{
		SensitiveFields: []string{
			"password", "token", "secret", "api_key",
			"client_secret", "authorization",
			// Специфичные для вашего API
			"stripe_key", "webhook_secret",
		},
		SensitivePatterns: []*regexp.Regexp{
			// Bearer tokens
			regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`),
			// API keys
			regexp.MustCompile(`(?i)(x-api-key:\s*)[a-zA-Z0-9\-_]{20,}`),
			// Webhook secrets
			regexp.MustCompile(`(?i)(whsec_)[a-zA-Z0-9]{32,}`),
		},
		Mask:        "[REDACTED]",
		MaxBodySize: 50 * 1024, // 50KB
	}

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		&httpclient.LoggingConfig{
			Logger:          logger,
			SanitizerConfig: sanitizerConfig,
			LogBody:         true,
		},
	)

	return &APIClient{
		client: &http.Client{
			Transport: rt,
			Timeout:   30 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// CreateUser пример метода с чувствительными данными
func (c *APIClient) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/users",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// BatchProcess пример обработки массива данных
func (c *APIClient) BatchProcess(ctx context.Context, items []BatchItem) (*BatchResponse, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/batch",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, err
	}

	return &batchResp, nil
}

// GetConfig пример с вложенным JSON
func (c *APIClient) GetConfig(ctx context.Context) (*Config, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"GET",
		c.baseURL+"/config",
		nil,
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Типы для примера
type CreateUserRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"` // Будет скрыт в логах
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Token     string    `json:"token"` // Будет скрыт в логах
	CreatedAt time.Time `json:"created_at"`
}

type BatchItem struct {
	ID     int                    `json:"id"`
	Data   map[string]interface{} `json:"data"`
	Secret string                 `json:"secret,omitempty"` // Будет скрыт
}

type BatchResponse struct {
	Processed int      `json:"processed"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

type Config struct {
	APIKey     string                 `json:"api_key"` // Будет скрыт
	WebhookURL string                 `json:"webhook_url"`
	Settings   map[string]interface{} `json:"settings"`
	SecretJSON string                 `json:"secret_json"` // Экранированный JSON
}

// Пример использования
func main() {
	// Создаем логгер (можно заменить на Zap, Logrus и т.д.)
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	// Создаем клиент
	client := NewAPIClient("https://api.example.com", "sk-test-key-12345", logger)

	ctx := context.Background()

	// Пример 1: Создание пользователя с чувствительными данными
	fmt.Println("=== Creating User ===")
	user, err := client.CreateUser(ctx, CreateUserRequest{
		Email:    "user@example.com",
		Password: "super-secret-password-123",
		Name:     "John Doe",
		Metadata: map[string]interface{}{
			"phone":      "+1-555-0123",
			"api_secret": "secret-value",
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Created user: %+v\n", user)
	}

	// Пример 2: Batch обработка с массивом
	fmt.Println("\n=== Batch Processing ===")
	items := []BatchItem{
		{
			ID: 1,
			Data: map[string]interface{}{
				"name":  "Item 1",
				"value": 100,
			},
			Secret: "secret-1",
		},
		{
			ID: 2,
			Data: map[string]interface{}{
				"name":  "Item 2",
				"value": 200,
			},
			Secret: "secret-2",
		},
	}

	batchResp, err := client.BatchProcess(ctx, items)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Batch response: %+v\n", batchResp)
	}

	// Пример 3: Получение конфигурации с вложенным JSON
	fmt.Println("\n=== Getting Config ===")
	config, err := client.GetConfig(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Config: %+v\n", config)
	}
}

// Пример middleware для добавления trace ID
type TracingRoundTripper struct {
	next http.RoundTripper
}

func NewTracingRoundTripper(next http.RoundTripper) *TracingRoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &TracingRoundTripper{next: next}
}

func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Добавляем trace ID из контекста
	if traceID := req.Context().Value("trace_id"); traceID != nil {
		req.Header.Set("X-Trace-ID", fmt.Sprint(traceID))
	}

	return t.next.RoundTrip(req)
}

// Пример комбинирования нескольких RoundTripper'ов
func CreateProductionClient(logger httpclient.Logger) *http.Client {
	// Базовый транспорт
	baseTransport := http.DefaultTransport

	// Добавляем tracing
	tracingTransport := NewTracingRoundTripper(baseTransport)

	// Добавляем логирование с санитизацией
	loggingTransport := httpclient.NewLoggingRoundTripper(
		tracingTransport,
		&httpclient.LoggingConfig{
			Logger:  logger,
			LogBody: true, // В проде можно поставить false для performance
		},
	)

	return &http.Client{
		Transport: loggingTransport,
		Timeout:   30 * time.Second,
	}
}

// Пример rate limiting RoundTripper (можно комбинировать)
type RateLimitingRoundTripper struct {
	next    http.RoundTripper
	limiter chan struct{}
}

func NewRateLimitingRoundTripper(next http.RoundTripper, rps int) *RateLimitingRoundTripper {
	limiter := make(chan struct{}, rps)

	// Заполняем канал
	for i := 0; i < rps; i++ {
		limiter <- struct{}{}
	}

	// Пополняем каждую секунду
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(rps))
		defer ticker.Stop()

		for range ticker.C {
			select {
			case limiter <- struct{}{}:
			default:
			}
		}
	}()

	return &RateLimitingRoundTripper{
		next:    next,
		limiter: limiter,
	}
}

func (r *RateLimitingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	<-r.limiter // Ждем доступный слот
	return r.next.RoundTrip(req)
}
