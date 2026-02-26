package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/alimzhanovlr/sdk/httpclient"
)

func main() {
	// Создаем логгер
	logger := httpclient.NewSimpleLogger(httpclient.DEBUG)

	// Пример 1: JSON (объект и массив)
	exampleJSON(logger)

	// Пример 2: XML
	exampleXML(logger)

	// Пример 3: Form URL-encoded
	exampleFormURLEncoded(logger)

	// Пример 4: Multipart Form
	exampleMultipartForm(logger)

	// Пример 5: JSON со строками (escaped JSON)
	exampleEscapedJSON(logger)

	// Пример 6: Большие тела и base64
	exampleLargeBodies(logger)

	// Пример 7: Кастомные правила
	exampleCustomRules(logger)

	// Пример 8: Headers санитизация
	exampleHeadersSanitization(logger)

	// Пример 9: Query параметры
	exampleQueryParameters(logger)
}

// Пример 1: JSON
func exampleJSON(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 1: JSON Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	// JSON объект
	fmt.Println("\n--- JSON Object ---")
	payload := map[string]interface{}{
		"username": "user@example.com",
		"password": "super-secret-123",
		"api_key":  "sk-1234567890abcdef",
		"user_data": map[string]interface{}{
			"name":  "John Doe",
			"token": "bearer-token-xyz",
		},
	}
	sendJSON(client, payload)

	// JSON массив
	fmt.Println("\n--- JSON Array ---")
	arrayPayload := []map[string]interface{}{
		{
			"id":       1,
			"password": "pass1",
			"token":    "tok1",
		},
		{
			"id":       2,
			"password": "pass2",
			"token":    "tok2",
		},
	}
	sendJSON(client, arrayPayload)
}

// Пример 2: XML
func exampleXML(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 2: XML Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<user>
    <username>john_doe</username>
    <password>secret-password-123</password>
    <api_key>sk-xml-key-12345</api_key>
    <profile token="bearer-token-abc">
        <name>John Doe</name>
        <email>john@example.com</email>
    </profile>
</user>`

	req, _ := http.NewRequest("POST", "https://api.example.com/xml", strings.NewReader(xmlBody))
	req.Header.Set("Content-Type", "application/xml")
	client.Do(req)
}

// Пример 3: Form URL-encoded
func exampleFormURLEncoded(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 3: Form URL-encoded Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	formData := "username=user@example.com&password=secret123&api_key=sk-form-key&remember=true"

	req, _ := http.NewRequest("POST", "https://api.example.com/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client.Do(req)
}

// Пример 4: Multipart Form
func exampleMultipartForm(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 4: Multipart Form Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	multipartBody := `------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="username"

john_doe
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="password"

secret-password-123
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="api_key"

sk-multipart-key-xyz
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="file"; filename="document.txt"
Content-Type: text/plain

This is file content
------WebKitFormBoundary7MA4YWxkTrZu0gW--`

	req, _ := http.NewRequest("POST", "https://api.example.com/upload", strings.NewReader(multipartBody))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW")
	client.Do(req)
}

// Пример 5: Escaped JSON (JSON в строке)
func exampleEscapedJSON(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 5: Escaped JSON in String")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	// JSON содержащий экранированный JSON в строке
	payload := map[string]interface{}{
		"config": `{"api_key":"sk-nested-123","secret":"my-secret-value"}`,
		"data": map[string]interface{}{
			"password": "outer-password",
		},
	}
	sendJSON(client, payload)
}

// Пример 6: Большие тела и base64
func exampleLargeBodies(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 6: Large Bodies and Base64")
	fmt.Println(strings.Repeat("=", 60))

	config := httpclient.DefaultLoggingConfig(logger)
	// Настраиваем более агрессивную обработку больших тел
	config.SanitizerConfig = &httpclient.SanitizerConfig{
		SensitiveFields:   httpclient.DefaultSanitizerConfig().SensitiveFields,
		SensitivePatterns: httpclient.DefaultSanitizerConfig().SensitivePatterns,
		Mask:              "***HIDDEN***",
		MaxBodySize:       5 * 1024, // Только 5KB для примера
		BodyRules: []httpclient.BodyProcessingRule{
			// Пропускаем base64 больше 500 байт
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 500 && looksLikeBase64(body)
				},
				Action:  httpclient.BodyActionSkip,
				Message: "[Base64 data detected - not logged]",
			},
			// Суммаризуем большие JSON
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 5*1024
				},
				Action: httpclient.BodyActionSummarize,
			},
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
	client := &http.Client{Transport: rt}

	// Большой JSON
	fmt.Println("\n--- Large JSON ---")
	largeData := make(map[string]interface{})
	for i := 0; i < 200; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = strings.Repeat("value", 10)
	}
	sendJSON(client, largeData)

	// Base64 данные
	fmt.Println("\n--- Base64 Data ---")
	base64Payload := map[string]interface{}{
		"filename": "image.png",
		"data":     strings.Repeat("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==", 10),
	}
	sendJSON(client, base64Payload)
}

// Пример 7: Кастомные правила
func exampleCustomRules(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 7: Custom Rules")
	fmt.Println(strings.Repeat("=", 60))

	config := &httpclient.LoggingConfig{
		Logger:          logger,
		LogRequestBody:  true,
		LogResponseBody: true,
		LogHeaders:      true,

		// Кастомное правило: не логировать body для определенных эндпоинтов
		ShouldLogBody: func(req *http.Request, contentType string, size int) bool {
			// Не логируем body для /upload эндпоинта
			if strings.Contains(req.URL.Path, "/upload") {
				return false
			}
			// Не логируем файлы
			if strings.HasPrefix(contentType, "image/") ||
				strings.HasPrefix(contentType, "application/pdf") {
				return false
			}
			return true
		},

		SanitizerConfig: &httpclient.SanitizerConfig{
			SensitiveFields: []string{
				"password", "token", "secret",
				// Добавляем специфичные для нашего API поля
				"internal_key", "webhook_url", "encryption_key",
			},
			SensitivePatterns: []*regexp.Regexp{
				// Кастомный паттерн для наших API ключей
				regexp.MustCompile(`(myapp-key-)[a-zA-Z0-9]{32}`),
				// Скрываем email адреса
				regexp.MustCompile(`([a-zA-Z0-9._%+-]+@)[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			},
			Mask:        "[REDACTED]",
			MaxBodySize: 10 * 1024,
		},
	}

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
	client := &http.Client{Transport: rt}

	payload := map[string]interface{}{
		"username":      "user@example.com",
		"internal_key":  "myapp-key-abcdef1234567890abcdef1234567890",
		"contact_email": "contact@company.com",
	}
	sendJSON(client, payload)
}

// Пример 8: Headers санитизация
func exampleHeadersSanitization(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 8: Headers Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	config := httpclient.DefaultLoggingConfig(logger)
	config.SanitizerConfig.HeaderMaskMode = httpclient.HeaderMaskPartial // Показываем часть

	rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
	req.Header.Set("Authorization", "Bearer sk-1234567890abcdefghijklmnopqrstuvwxyz")
	req.Header.Set("X-API-Key", "api-key-1234567890")
	req.Header.Set("Cookie", "session=abc123; user_id=456")
	req.Header.Set("User-Agent", "MyApp/1.0")
	req.Header.Set("Accept", "application/json")

	client.Do(req)
}

// Пример 9: Query параметры
func exampleQueryParameters(logger httpclient.Logger) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 9: Query Parameters Sanitization")
	fmt.Println(strings.Repeat("=", 60))

	rt := httpclient.NewLoggingRoundTripper(
		http.DefaultTransport,
		httpclient.DefaultLoggingConfig(logger),
	)
	client := &http.Client{Transport: rt}

	// URL с чувствительными параметрами
	req, _ := http.NewRequest(
		"GET",
		"https://api.example.com/users?user_id=123&api_key=sk-secret-key&token=bearer-xyz&limit=10",
		nil,
	)

	client.Do(req)
}

// Вспомогательные функции

func sendJSON(client *http.Client, payload interface{}) {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.example.com/endpoint", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	client.Do(req)
}

func looksLikeBase64(body []byte) bool {
	if len(body) < 100 {
		return false
	}

	base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

	sample := body
	if len(body) > 1000 {
		sample = body[:1000]
	}

	validChars := 0
	for _, b := range sample {
		if strings.ContainsRune(base64Chars, rune(b)) || b == '\n' || b == '\r' {
			validChars++
		}
	}

	return float64(validChars)/float64(len(sample)) > 0.9
}
