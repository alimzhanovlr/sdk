package httpclient

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizer_JSONObject(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:        "simple password field",
			input:       `{"username":"user","password":"secret123"}`,
			contains:    []string{"username", "user"},
			notContains: []string{"secret123"},
		},
		{
			name:        "nested sensitive fields",
			input:       `{"user":{"name":"John","credentials":{"password":"pass","api_key":"key123"}}}`,
			contains:    []string{"John"},
			notContains: []string{"pass", "key123"},
		},
		{
			name:        "mixed case sensitive fields",
			input:       `{"Password":"secret","API_KEY":"key","Token":"token123"}`,
			notContains: []string{"secret", "key", "token123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.Sanitize([]byte(tt.input), "application/json")

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q, but it doesn't. Result: %s", want, result)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("Expected result NOT to contain %q, but it does. Result: %s", notWant, result)
				}
			}
		})
	}
}

func TestSanitizer_JSONArray(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	input := `[{"id":1,"token":"tok1"},{"id":2,"token":"tok2"}]`
	result := sanitizer.Sanitize([]byte(input), "application/json")

	// Проверяем что это валидный JSON массив
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("Result is not valid JSON array: %v", err)
	}

	// Проверяем что токены скрыты
	if strings.Contains(result, "tok1") || strings.Contains(result, "tok2") {
		t.Errorf("Tokens should be sanitized. Result: %s", result)
	}

	// Проверяем что ID остались
	if !strings.Contains(result, `"id"`) {
		t.Errorf("IDs should be preserved. Result: %s", result)
	}
}

func TestSanitizer_EscapedJSON(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	// JSON строка содержащая экранированный JSON
	input := `{"config":"{\"api_key\":\"sk-123\",\"secret\":\"mysecret\"}"}`
	result := sanitizer.Sanitize([]byte(input), "application/json")

	// Основной JSON должен быть валиден
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Чувствительные данные должны быть скрыты
	if strings.Contains(result, "sk-123") {
		t.Errorf("API key should be sanitized in escaped JSON. Result: %s", result)
	}
}

func TestSanitizer_PlainText(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	tests := []struct {
		name        string
		input       string
		notContains []string
	}{
		{
			name:        "bearer token",
			input:       "Authorization: Bearer sk-1234567890abcdef",
			notContains: []string{"sk-1234567890abcdef"},
		},
		{
			name:        "api key pattern",
			input:       "api_key: abcdef1234567890123456789012",
			notContains: []string{"abcdef1234567890123456789012"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.Sanitize([]byte(tt.input), "text/plain")

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("Expected result NOT to contain %q. Result: %s", notWant, result)
				}
			}
		})
	}
}

func TestSanitizer_MaxBodySize(t *testing.T) {
	config := &SanitizerConfig{
		SensitiveFields: []string{"password"},
		Mask:            "***",
		MaxBodySize:     50, // Очень маленький лимит для теста
	}
	sanitizer := NewSanitizer(config)

	largeBody := strings.Repeat("a", 1000)
	result := sanitizer.Sanitize([]byte(largeBody), "text/plain")

	if len(result) > 200 { // С учетом сообщения о truncate
		t.Errorf("Body should be truncated. Length: %d", len(result))
	}

	if !strings.Contains(result, "truncated") {
		t.Errorf("Result should indicate truncation. Result: %s", result)
	}
}

func TestSanitizer_EmptyBody(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	result := sanitizer.Sanitize([]byte{}, "application/json")
	if result != "" {
		t.Errorf("Empty body should return empty string, got: %q", result)
	}
}

func TestSanitizer_NonJSONContent(t *testing.T) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())

	tests := []struct {
		name        string
		input       string
		contentType string
	}{
		{
			name:        "XML content",
			input:       `<user><password>secret</password></user>`,
			contentType: "application/xml",
		},
		{
			name:        "HTML content",
			input:       `<html><body>password=secret</body></html>`,
			contentType: "text/html",
		},
		{
			name:        "form data",
			input:       `username=user&password=secret&token=abc123`,
			contentType: "application/x-www-form-urlencoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.Sanitize([]byte(tt.input), tt.contentType)
			// Просто проверяем что не падает
			if result == "" {
				t.Errorf("Result should not be empty for non-JSON content")
			}
		})
	}
}

func TestSanitizer_CustomFields(t *testing.T) {
	config := &SanitizerConfig{
		SensitiveFields: []string{"ssn", "credit_card", "user_secret"},
		Mask:            "[HIDDEN]",
		MaxBodySize:     10 * 1024,
	}
	sanitizer := NewSanitizer(config)

	input := `{"ssn":"123-45-6789","credit_card":"4111111111111111","name":"John"}`
	result := sanitizer.Sanitize([]byte(input), "application/json")

	if strings.Contains(result, "123-45-6789") {
		t.Errorf("SSN should be sanitized")
	}
	if strings.Contains(result, "4111111111111111") {
		t.Errorf("Credit card should be sanitized")
	}
	if !strings.Contains(result, "John") {
		t.Errorf("Name should be preserved")
	}
	if !strings.Contains(result, "[HIDDEN]") {
		t.Errorf("Custom mask should be used")
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/vnd.api+json", true},
		{"text/json", true},
		{"text/plain", false},
		{"application/xml", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isJSON(tt.contentType)
		if result != tt.expected {
			t.Errorf("isJSON(%q) = %v, want %v", tt.contentType, result, tt.expected)
		}
	}
}

func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		body     string
		expected bool
	}{
		{`{"key":"value"}`, true},
		{`[1,2,3]`, true},
		{`  {"key":"value"}  `, true},
		{`  [1,2,3]  `, true},
		{`not json`, false},
		{``, false},
		{`<xml></xml>`, false},
		{`{"incomplete"`, false},
	}

	for _, tt := range tests {
		result := looksLikeJSON(tt.body)
		if result != tt.expected {
			t.Errorf("looksLikeJSON(%q) = %v, want %v", tt.body, result, tt.expected)
		}
	}
}
