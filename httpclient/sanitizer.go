package httpclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// BodyProcessingRule правило обработки body
type BodyProcessingRule struct {
	// Условие для применения правила
	Condition func(contentType string, body []byte, size int) bool
	// Действие: skip, truncate, summarize, sanitize
	Action BodyAction
	// Сообщение при пропуске
	Message string
}

type BodyAction string

const (
	BodyActionSkip      BodyAction = "skip"      // Полностью пропустить логирование
	BodyActionTruncate  BodyAction = "truncate"  // Обрезать до MaxBodySize
	BodyActionSummarize BodyAction = "summarize" // Показать только метаданные
	BodyActionSanitize  BodyAction = "sanitize"  // Санитизировать и показать
)

// SanitizerConfig расширенная конфигурация санитайзера
type SanitizerConfig struct {
	// Поля для скрытия в JSON/XML/Form (case-insensitive)
	SensitiveFields []string

	// Regex паттерны для поиска в любом тексте
	SensitivePatterns []*regexp.Regexp

	// Маска для замены
	Mask string

	// Максимальный размер body для логирования (байты)
	MaxBodySize int

	// Правила обработки body (применяются по порядку)
	BodyRules []BodyProcessingRule

	// Скрывать ли значения заголовков целиком или только часть
	HeaderMaskMode HeaderMaskMode

	// Кастомные заголовки для санитизации (дополнительно к дефолтным)
	SensitiveHeaders []string
}

type HeaderMaskMode string

const (
	HeaderMaskFull    HeaderMaskMode = "full"    // Полностью скрыть значение
	HeaderMaskPartial HeaderMaskMode = "partial" // Показать первые/последние символы
)

// DefaultSanitizerConfig дефолтная конфигурация с расширенными правилами
func DefaultSanitizerConfig() *SanitizerConfig {
	return &SanitizerConfig{
		SensitiveFields: []string{
			// Аутентификация
			"password", "passwd", "pwd", "secret", "token",
			"api_key", "apikey", "api_secret", "access_token", "refresh_token",
			"client_secret", "client_id", "authorization", "auth",
			"bearer", "session", "session_id", "cookie",

			// Персональные данные
			"ssn", "social_security", "passport", "driver_license",
			"tax_id", "ein", "vat",

			// Финансовые данные
			"credit_card", "card_number", "card_num", "cvv", "cvc",
			"pin", "account_number", "routing_number", "iban", "swift",

			// Криптография
			"private_key", "public_key", "encryption_key", "signing_key",
			"certificate", "cert", "key", "pem",

			// Специфичные сервисы
			"stripe_key", "aws_secret", "gcp_key", "azure_key",
			"webhook_secret", "signing_secret",
		},

		SensitivePatterns: []*regexp.Regexp{
			// Bearer tokens
			regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`),

			// API keys (различные форматы)
			regexp.MustCompile(`(?i)(api[_-]?key["']?\s*[:=]\s*["']?)[a-zA-Z0-9\-_]{20,}`),
			regexp.MustCompile(`(?i)(x-api-key:\s*)[a-zA-Z0-9\-_]{20,}`),

			// AWS ключи
			regexp.MustCompile(`(AKIA[0-9A-Z]{16})`),
			regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?access[_-]?key["']?\s*[:=]\s*["']?)([a-zA-Z0-9/+=]{40})`),

			// Google API keys
			regexp.MustCompile(`(AIza[0-9A-Za-z\-_]{35})`),

			// GitHub tokens
			regexp.MustCompile(`(gh[ps]_[a-zA-Z0-9]{36})`),

			// JWT токены
			regexp.MustCompile(`(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`),

			// Private keys (начало)
			regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),

			// Email (опционально - может быть не сенситивным)
			// regexp.MustCompile(`([a-zA-Z0-9._%+-]+@)[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),

			// Credit card numbers
			regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`),
		},

		Mask:        "***REDACTED***",
		MaxBodySize: 100 * 1024, // 100KB

		BodyRules: []BodyProcessingRule{
			// Правило 1: Пропускаем бинарные файлы
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return isBinaryContent(contentType)
				},
				Action:  BodyActionSkip,
				Message: "[Binary content - not logged]",
			},

			// Правило 2: Пропускаем base64 данные больше 1KB
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 1024 && looksLikeBase64(body)
				},
				Action:  BodyActionSkip,
				Message: "[Base64 encoded data - not logged]",
			},

			// Правило 3: Суммаризуем очень большие JSON/XML
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 500*1024 && (isJSON(contentType) || isXML(contentType))
				},
				Action:  BodyActionSummarize,
				Message: "", // Будет сгенерировано автоматически
			},

			// Правило 4: Truncate для больших тел
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 100*1024
				},
				Action: BodyActionTruncate,
			},
		},

		HeaderMaskMode: HeaderMaskPartial,
		SensitiveHeaders: []string{
			"authorization", "proxy-authorization",
			"cookie", "set-cookie",
			"x-api-key", "x-auth-token", "x-access-token",
			"api-key", "apikey",
		},
	}
}

// Sanitizer расширенный санитайзер
type Sanitizer struct {
	config *SanitizerConfig
}

// NewSanitizer создает санитайзер
func NewSanitizer(config *SanitizerConfig) *Sanitizer {
	if config == nil {
		config = DefaultSanitizerConfig()
	}

	// Дополняем дефолтными заголовками если не заданы
	if len(config.SensitiveHeaders) == 0 {
		config.SensitiveHeaders = DefaultSanitizerConfig().SensitiveHeaders
	}

	return &Sanitizer{config: config}
}

// SanitizeBody очищает тело запроса/ответа
func (s *Sanitizer) SanitizeBody(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	size := len(body)

	// Применяем правила обработки
	for _, rule := range s.config.BodyRules {
		if rule.Condition(contentType, body, size) {
			switch rule.Action {
			case BodyActionSkip:
				if rule.Message != "" {
					return rule.Message
				}
				return "[Body not logged]"

			case BodyActionSummarize:
				return s.summarizeBody(body, contentType, size)

			case BodyActionTruncate:
				return s.truncateBody(body, contentType)

			case BodyActionSanitize:
				// Продолжаем обработку
			}
		}
	}

	// Определяем формат и санитизируем
	if isJSON(contentType) || looksLikeJSON(string(body)) {
		return s.sanitizeJSON(string(body))
	}

	if isXML(contentType) || looksLikeXML(string(body)) {
		return s.sanitizeXML(string(body))
	}

	if isFormURLEncoded(contentType) {
		return s.sanitizeFormURLEncoded(string(body))
	}

	if isMultipartForm(contentType) {
		return s.sanitizeMultipartForm(string(body))
	}

	// Обрабатываем как обычный текст
	return s.sanitizeText(string(body))
}

// SanitizeHeaders очищает заголовки
func (s *Sanitizer) SanitizeHeaders(headers map[string][]string) map[string]string {
	result := make(map[string]string)

	for key, values := range headers {
		if s.isSensitiveHeader(key) {
			result[key] = s.maskHeaderValue(values)
		} else {
			result[key] = strings.Join(values, ", ")
		}
	}

	return result
}

// sanitizeJSON обрабатывает JSON
func (s *Sanitizer) sanitizeJSON(body string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return s.sanitizeText(body)
	}

	sanitized := s.sanitizeValue(data)
	result, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return s.sanitizeText(body)
	}

	return string(result)
}

// sanitizeXML обрабатывает XML
func (s *Sanitizer) sanitizeXML(body string) string {
	// Простая санитизация XML через regex
	// Для более сложных случаев можно распарсить через xml.Unmarshal
	result := body

	// Ищем теги с чувствительными данными
	for _, field := range s.config.SensitiveFields {
		// <password>value</password> -> <password>***</password>
		pattern := regexp.MustCompile(`(?i)(<` + regexp.QuoteMeta(field) + `[^>]*>)([^<]+)(</` + regexp.QuoteMeta(field) + `>)`)
		result = pattern.ReplaceAllString(result, "${1}"+s.config.Mask+"${3}")

		// <tag password="value"> -> <tag password="***">
		attrPattern := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(field) + `\s*=\s*["'])([^"']+)(["'])`)
		result = attrPattern.ReplaceAllString(result, "${1}"+s.config.Mask+"${3}")
	}

	// Применяем паттерны
	for _, pattern := range s.config.SensitivePatterns {
		result = pattern.ReplaceAllString(result, "$1"+s.config.Mask)
	}

	return result
}

// sanitizeFormURLEncoded обрабатывает application/x-www-form-urlencoded
func (s *Sanitizer) sanitizeFormURLEncoded(body string) string {
	values, err := url.ParseQuery(body)
	if err != nil {
		return s.sanitizeText(body)
	}

	sanitized := url.Values{}
	for key, vals := range values {
		if s.isSensitiveField(key) {
			sanitized[key] = []string{s.config.Mask}
		} else {
			// Проверяем значения на паттерны
			newVals := make([]string, len(vals))
			for i, val := range vals {
				newVals[i] = s.sanitizeText(val)
			}
			sanitized[key] = newVals
		}
	}

	return sanitized.Encode()
}

// sanitizeMultipartForm обрабатывает multipart/form-data
func (s *Sanitizer) sanitizeMultipartForm(body string) string {
	// Multipart сложнее, делаем упрощенную обработку
	lines := strings.Split(body, "\n")
	result := make([]string, 0, len(lines))

	inSensitiveField := false
	currentFieldName := ""

	for _, line := range lines {
		// Ищем Content-Disposition с именем поля
		if strings.Contains(line, "Content-Disposition") {
			nameMatch := regexp.MustCompile(`name="([^"]+)"`).FindStringSubmatch(line)
			if len(nameMatch) > 1 {
				currentFieldName = nameMatch[1]
				inSensitiveField = s.isSensitiveField(currentFieldName)
			}
			result = append(result, line)
			continue
		}

		// Если это граница (boundary)
		if strings.HasPrefix(line, "--") {
			inSensitiveField = false
			currentFieldName = ""
			result = append(result, line)
			continue
		}

		// Если в чувствительном поле - заменяем значение
		if inSensitiveField && line != "" && !strings.HasPrefix(line, "Content-") {
			result = append(result, s.config.Mask)
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// sanitizeValue рекурсивно обрабатывает JSON значения
func (s *Sanitizer) sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if s.isSensitiveField(key) {
				result[key] = s.config.Mask
			} else {
				result[key] = s.sanitizeValue(val)
			}
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = s.sanitizeValue(val)
		}
		return result

	case string:
		// Проверяем на вложенный JSON
		if looksLikeJSON(v) {
			nested := s.sanitizeJSON(v)
			return nested
		}
		return s.sanitizeText(v)

	default:
		return v
	}
}

// sanitizeText обрабатывает текст
func (s *Sanitizer) sanitizeText(text string) string {
	result := text

	for _, pattern := range s.config.SensitivePatterns {
		result = pattern.ReplaceAllString(result, "$1"+s.config.Mask)
	}

	return result
}

// isSensitiveField проверяет чувствительность поля
func (s *Sanitizer) isSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, sensitive := range s.config.SensitiveFields {
		if strings.Contains(lower, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// isSensitiveHeader проверяет чувствительность заголовка
func (s *Sanitizer) isSensitiveHeader(headerName string) bool {
	lower := strings.ToLower(headerName)
	for _, sensitive := range s.config.SensitiveHeaders {
		if strings.ToLower(sensitive) == lower {
			return true
		}
	}
	return false
}

// maskHeaderValue маскирует значение заголовка
func (s *Sanitizer) maskHeaderValue(values []string) string {
	if len(values) == 0 {
		return ""
	}

	value := strings.Join(values, ", ")

	if s.config.HeaderMaskMode == HeaderMaskFull {
		return s.config.Mask
	}

	// Partial - показываем первые и последние символы
	if len(value) <= 8 {
		return s.config.Mask
	}

	return value[:4] + s.config.Mask + value[len(value)-4:]
}

// truncateBody обрезает тело
func (s *Sanitizer) truncateBody(body []byte, contentType string) string {
	maxSize := s.config.MaxBodySize
	if len(body) <= maxSize {
		return s.SanitizeBody(body, contentType)
	}

	// Пытаемся обрезать умно
	truncated := body[:maxSize]
	result := string(truncated)

	return result + "\n... [truncated, total: " + formatSize(len(body)) + "]"
}

// summarizeBody создает сводку для большого тела
func (s *Sanitizer) summarizeBody(body []byte, contentType string, size int) string {
	summary := "[Large body - " + formatSize(size) + "]"

	if isJSON(contentType) {
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			switch v := data.(type) {
			case map[string]interface{}:
				summary += " Object with " + formatInt(len(v)) + " keys"
			case []interface{}:
				summary += " Array with " + formatInt(len(v)) + " items"
			}
		}
	}

	if isXML(contentType) {
		summary += " XML document"
	}

	return summary
}

// Вспомогательные функции

func isJSON(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/vnd.api+json") ||
		strings.Contains(ct, "text/json") ||
		strings.HasSuffix(ct, "+json")
}

func isXML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/xml") ||
		strings.Contains(ct, "text/xml") ||
		strings.HasSuffix(ct, "+xml")
}

func isFormURLEncoded(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded")
}

func isMultipartForm(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "multipart/form-data")
}

func isBinaryContent(contentType string) bool {
	ct := strings.ToLower(contentType)
	binaryTypes := []string{
		"application/octet-stream",
		"application/pdf",
		"image/", "audio/", "video/",
		"application/zip", "application/gzip",
		"application/x-tar",
	}

	for _, bt := range binaryTypes {
		if strings.Contains(ct, bt) {
			return true
		}
	}
	return false
}

func looksLikeJSON(body string) bool {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

func looksLikeXML(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">")
}

func looksLikeBase64(body []byte) bool {
	if len(body) < 100 {
		return false
	}

	// Base64 содержит только определенные символы
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

	// Если больше 90% символов валидны для base64
	return float64(validChars)/float64(len(sample)) > 0.9
}

func formatSize(size int) string {
	if size < 1024 {
		return formatInt(size) + " bytes"
	}
	if size < 1024*1024 {
		return formatInt(size/1024) + " KB"
	}
	return formatInt(size/(1024*1024)) + " MB"
}

func formatInt(n int) string {
	return strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%d", n), ",", ""), ".", ",")
}
