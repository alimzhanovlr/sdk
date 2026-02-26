package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/alimzhanovlr/sdk/httpclient"
)

func main() {
	// Пример 1: Базовое использование
	basicExample()

	// Пример 2: Кастомная конфигурация
	customConfigExample()

	// Пример 3: Разные типы данных
	differentContentTypesExample()
}

// basicExample базовый пример
func basicExample() {
	fmt.Println("=== Basic Example ===")

	// Создаем логгер
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	// Создаем RoundTripper с дефолтными настройками
	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		&httpclient.LoggingConfig{
			Logger:  logger,
			LogBody: true,
		},
	)

	// Создаем HTTP клиент
	client := &http.Client{
		Transport: rt,
		Timeout:   30 * time.Second,
	}

	// Отправляем запрос с чувствительными данными
	payload := map[string]interface{}{
		"username": "user@example.com",
		"password": "super-secret-password",
		"api_key":  "sk-1234567890abcdef",
		"data": map[string]string{
			"name": "John",
			"age":  "30",
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.example.com/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token-12345")

	client.Do(req)
}

// customConfigExample пример с кастомной конфигурацией
func customConfigExample() {
	fmt.Println("\n=== Custom Config Example ===")

	logger := httpclient.NewSimpleLogger(httpclient.INFO)

	// Кастомная конфигурация санитайзера
	sanitizerConfig := &httpclient.SanitizerConfig{
		SensitiveFields: []string{
			"password", "token", "secret", "ssn",
			"credit_card", "private_key", "session_id",
		},
		SensitivePatterns: []*regexp.Regexp{
			// Email адреса
			regexp.MustCompile(`([a-zA-Z0-9._%+-]+@)[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			// Номера телефонов
			regexp.MustCompile(`(\+?1[-.]?)?\(?([0-9]{3})\)?[-.]?([0-9]{3})[-.]?([0-9]{4})`),
			// Bearer tokens
			regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`),
		},
		Mask:        "[HIDDEN]",
		MaxBodySize: 5 * 1024, // 5KB лимит
	}

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		&httpclient.LoggingConfig{
			Logger:          logger,
			SanitizerConfig: sanitizerConfig,
			LogBody:         true,
		},
	)

	client := &http.Client{Transport: rt}

	// Запрос с email и телефоном
	payload := map[string]string{
		"email":   "user@example.com",
		"phone":   "+1-555-123-4567",
		"message": "Contact me at john.doe@example.com or call +1-555-987-6543",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.example.com/contact", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client.Do(req)
}

// differentContentTypesExample примеры с разными типами контента
func differentContentTypesExample() {
	fmt.Println("\n=== Different Content Types Example ===")

	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)
	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		&httpclient.LoggingConfig{
			Logger:  logger,
			LogBody: true,
		},
	)
	client := &http.Client{Transport: rt}

	// 1. JSON объект
	fmt.Println("\n--- JSON Object ---")
	jsonObject := map[string]interface{}{
		"username": "test",
		"password": "secret123",
	}
	sendRequest(client, "POST", "https://api.example.com/json-object", jsonObject, "application/json")

	// 2. JSON массив
	fmt.Println("\n--- JSON Array ---")
	jsonArray := []map[string]interface{}{
		{"id": 1, "token": "token1"},
		{"id": 2, "token": "token2"},
	}
	sendRequest(client, "POST", "https://api.example.com/json-array", jsonArray, "application/json")

	// 3. Вложенный JSON с экранированными строками
	fmt.Println("\n--- Nested JSON with escaped strings ---")
	nestedJSON := map[string]interface{}{
		"data": map[string]interface{}{
			"config":   `{\"api_key\":\"sk-123456\",\"secret\":\"my-secret\"}`,
			"password": "secret-pass",
		},
	}
	sendRequest(client, "POST", "https://api.example.com/nested", nestedJSON, "application/json")

	// 4. Plain text с чувствительными данными
	fmt.Println("\n--- Plain Text ---")
	plainText := "Authorization: Bearer sk-1234567890\nAPI-Key: secret-key-value"
	sendTextRequest(client, "POST", "https://api.example.com/text", plainText, "text/plain")

	// 5. URL-encoded form data
	fmt.Println("\n--- Form Data ---")
	formData := "username=user&password=secret123&api_key=sk-abcdef"
	sendTextRequest(client, "POST", "https://api.example.com/form", formData, "application/x-www-form-urlencoded")

	// 6. Большое тело (будет обрезано)
	fmt.Println("\n--- Large Body ---")
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	sendRequest(client, "POST", "https://api.example.com/large", largeData, "application/json")
}

func sendRequest(client *http.Client, method, url string, payload interface{}, contentType string) {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", contentType)
	client.Do(req)
}

func sendTextRequest(client *http.Client, method, url, text, contentType string) {
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(text))
	req.Header.Set("Content-Type", contentType)
	client.Do(req)
}

// Пример интеграции со сторонним логгером (zap, logrus, etc)
type ZapLoggerAdapter struct {
	// logger *zap.Logger
}

func (z *ZapLoggerAdapter) Debug(msg string, fields ...interface{}) {
	// z.logger.Debug(msg, convertFields(fields)...)
	fmt.Printf("ZAP DEBUG: %s %v\n", msg, fields)
}

func (z *ZapLoggerAdapter) Info(msg string, fields ...interface{}) {
	// z.logger.Info(msg, convertFields(fields)...)
	fmt.Printf("ZAP INFO: %s %v\n", msg, fields)
}

func (z *ZapLoggerAdapter) Error(msg string, fields ...interface{}) {
	// z.logger.Error(msg, convertFields(fields)...)
	fmt.Printf("ZAP ERROR: %s %v\n", msg, fields)
}
