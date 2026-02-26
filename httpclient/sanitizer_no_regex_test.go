package httpclient

import (
	"regexp"
	"strings"
	"testing"
)

// Тестовые данные
var (
	testJSONSmall = `{"username":"user","password":"secret123","api_key":"sk-1234567890"}`

	testJSONLarge = func() string {
		var sb strings.Builder
		sb.WriteString(`{"users":[`)
		for i := 0; i < 100; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`{"id":`)
			sb.WriteString(string(rune(i)))
			sb.WriteString(`,"password":"secret123","token":"tok_`)
			sb.WriteString(string(rune(i)))
			sb.WriteString(`"}`)
		}
		sb.WriteString(`]}`)
		return sb.String()
	}()

	testTextWithTokens = `Authorization: Bearer sk-1234567890abcdefghijklmnop
X-API-Key: api-key-abcdefghijklmnopqrstuvwxyz123456
JWT: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
AWS_KEY: AKIAIOSFODNN7EXAMPLE
Credit Card: 4532-1488-0343-6467`

	testXML = `<?xml version="1.0"?>
<user>
	<username>john</username>
	<password>secret123</password>
	<api_key>sk-key-xyz</api_key>
	<token>bearer-token-abc</token>
</user>`

	testForm = "username=user&password=secret123&api_key=sk-abcdef&email=user@example.com"
)

// ====================================================================================
// БЕНЧМАРКИ: JSON
// ====================================================================================

func BenchmarkJSON_WithRegex_Small(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testJSONSmall), "application/json")
	}
}

func BenchmarkJSON_NoRegex_Small(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testJSONSmall), "application/json")
	}
}

func BenchmarkJSON_WithRegex_Large(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testJSONLarge), "application/json")
	}
}

func BenchmarkJSON_NoRegex_Large(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testJSONLarge), "application/json")
	}
}

// ====================================================================================
// БЕНЧМАРКИ: Plain Text (с токенами)
// ====================================================================================

func BenchmarkText_WithRegex(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testTextWithTokens), "text/plain")
	}
}

func BenchmarkText_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testTextWithTokens), "text/plain")
	}
}

// ====================================================================================
// БЕНЧМАРКИ: XML
// ====================================================================================

func BenchmarkXML_WithRegex(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testXML), "application/xml")
	}
}

func BenchmarkXML_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testXML), "application/xml")
	}
}

// ====================================================================================
// БЕНЧМАРКИ: Form Data
// ====================================================================================

func BenchmarkForm_WithRegex(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testForm), "application/x-www-form-urlencoded")
	}
}

func BenchmarkForm_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testForm), "application/x-www-form-urlencoded")
	}
}

// ====================================================================================
// БЕНЧМАРКИ: Отдельные детекторы
// ====================================================================================

func BenchmarkBearerToken_Regex(b *testing.B) {
	pattern := regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`)
	text := "Authorization: Bearer sk-1234567890abcdefghijklmnop"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pattern.ReplaceAllString(text, "$1***")
	}
}

func BenchmarkBearerToken_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	text := "Authorization: Bearer sk-1234567890abcdefghijklmnop"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.hideBearerTokens(text)
	}
}

func BenchmarkJWT_Regex(b *testing.B) {
	pattern := regexp.MustCompile(`(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`)
	text := "JWT: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pattern.ReplaceAllString(text, "***")
	}
}

func BenchmarkJWT_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	text := "JWT: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.hideJWTTokens(text)
	}
}

func BenchmarkCreditCard_Regex(b *testing.B) {
	pattern := regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14})\b`)
	text := "Card: 4532-1488-0343-6467"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pattern.ReplaceAllString(text, "***")
	}
}

func BenchmarkCreditCard_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	text := "Card: 4532-1488-0343-6467"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.hideCreditCards(text)
	}
}

// ====================================================================================
// БЕНЧМАРКИ: Memory Allocation
// ====================================================================================

func BenchmarkAlloc_WithRegex(b *testing.B) {
	sanitizer := NewSanitizer(DefaultSanitizerConfig())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testTextWithTokens), "text/plain")
	}
}

func BenchmarkAlloc_NoRegex(b *testing.B) {
	sanitizer := NewSanitizerNoRegex(DefaultSanitizerConfigNoRegex())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizer.SanitizeBody([]byte(testTextWithTokens), "text/plain")
	}
}
