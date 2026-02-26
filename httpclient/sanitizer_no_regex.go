package httpclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// SanitizerConfigNoRegex конфигурация без regex
type SanitizerConfigNoRegex struct {
	SensitiveFields  []string
	Mask             string
	MaxBodySize      int
	BodyRules        []BodyProcessingRule
	HeaderMaskMode   HeaderMaskMode
	SensitiveHeaders []string

	// Вместо regex - простые string матчеры
	EnableBearerTokenDetection bool
	EnableAPIKeyDetection      bool
	EnableJWTDetection         bool
	EnableCreditCardDetection  bool
	EnableEmailDetection       bool
	EnableAWSKeyDetection      bool
}

// DefaultSanitizerConfigNoRegex дефолтная конфигурация без regex
func DefaultSanitizerConfigNoRegex() *SanitizerConfigNoRegex {
	return &SanitizerConfigNoRegex{
		SensitiveFields: []string{
			"password", "passwd", "pwd", "secret", "token",
			"api_key", "apikey", "api_secret", "access_token", "refresh_token",
			"client_secret", "authorization", "auth",
			"bearer", "session", "session_id", "cookie",
			"ssn", "credit_card", "card_number", "cvv", "cvc",
			"private_key", "encryption_key",
		},
		Mask:        "***REDACTED***",
		MaxBodySize: 100 * 1024,
		BodyRules: []BodyProcessingRule{
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return isBinaryContent(contentType)
				},
				Action:  BodyActionSkip,
				Message: "[Binary content - not logged]",
			},
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 1024 && looksLikeBase64(body)
				},
				Action:  BodyActionSkip,
				Message: "[Base64 encoded data - not logged]",
			},
			{
				Condition: func(contentType string, body []byte, size int) bool {
					return size > 100*1024
				},
				Action: BodyActionTruncate,
			},
		},
		HeaderMaskMode:             HeaderMaskPartial,
		EnableBearerTokenDetection: true,
		EnableAPIKeyDetection:      true,
		EnableJWTDetection:         true,
		EnableCreditCardDetection:  true,
		EnableAWSKeyDetection:      true,
	}
}

// SanitizerNoRegex санитайзер без regex
type SanitizerNoRegex struct {
	config *SanitizerConfigNoRegex
}

// NewSanitizerNoRegex создает санитайзер без regex
func NewSanitizerNoRegex(config *SanitizerConfigNoRegex) *SanitizerNoRegex {
	if config == nil {
		config = DefaultSanitizerConfigNoRegex()
	}
	return &SanitizerNoRegex{config: config}
}

// SanitizeBody очищает body без использования regex
func (s *SanitizerNoRegex) SanitizeBody(body []byte, contentType string) string {
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
			}
		}
	}

	// Определяем формат
	if isJSON(contentType) || looksLikeJSON(string(body)) {
		return s.sanitizeJSON(string(body))
	}

	if isXML(contentType) || looksLikeXML(string(body)) {
		return s.sanitizeXML(string(body))
	}

	if isFormURLEncoded(contentType) {
		return s.sanitizeFormURLEncoded(string(body))
	}

	// Plain text - применяем детекторы без regex
	return s.sanitizeText(string(body))
}

// sanitizeJSON обрабатывает JSON
func (s *SanitizerNoRegex) sanitizeJSON(body string) string {
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

// sanitizeValue рекурсивно обрабатывает значения
func (s *SanitizerNoRegex) sanitizeValue(value interface{}) interface{} {
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
		if looksLikeJSON(v) {
			return s.sanitizeJSON(v)
		}
		return s.sanitizeText(v)

	default:
		return v
	}
}

// sanitizeXML обрабатывает XML простым поиском
func (s *SanitizerNoRegex) sanitizeXML(body string) string {
	result := body

	// Для каждого чувствительного поля
	for _, field := range s.config.SensitiveFields {
		// Ищем <field>value</field>
		result = s.replaceXMLTag(result, field)
		// Ищем атрибуты field="value"
		result = s.replaceXMLAttribute(result, field)
	}

	// Применяем детекторы
	result = s.sanitizeText(result)

	return result
}

// replaceXMLTag заменяет содержимое XML тега
func (s *SanitizerNoRegex) replaceXMLTag(text, fieldName string) string {
	openTag := "<" + fieldName + ">"
	closeTag := "</" + fieldName + ">"

	result := text
	for {
		start := strings.Index(result, openTag)
		if start == -1 {
			// Пробуем case-insensitive
			start = indexCaseInsensitive(result, openTag)
			if start == -1 {
				break
			}
		}

		end := strings.Index(result[start:], closeTag)
		if end == -1 {
			break
		}

		// Заменяем содержимое между тегами
		beforeValue := result[:start+len(openTag)]
		afterValue := result[start+end:]
		result = beforeValue + s.config.Mask + afterValue
	}

	return result
}

// replaceXMLAttribute заменяет значения XML атрибутов
func (s *SanitizerNoRegex) replaceXMLAttribute(text, fieldName string) string {
	result := text

	// Ищем field="value" или field='value'
	for _, quote := range []string{`"`, `'`} {
		pattern := fieldName + "=" + quote

		for {
			start := indexCaseInsensitive(result, pattern)
			if start == -1 {
				break
			}

			valueStart := start + len(pattern)
			valueEnd := strings.Index(result[valueStart:], quote)
			if valueEnd == -1 {
				break
			}

			// Заменяем значение
			before := result[:valueStart]
			after := result[valueStart+valueEnd:]
			result = before + s.config.Mask + after
		}
	}

	return result
}

// sanitizeFormURLEncoded обрабатывает form data
func (s *SanitizerNoRegex) sanitizeFormURLEncoded(body string) string {
	values, err := url.ParseQuery(body)
	if err != nil {
		return s.sanitizeText(body)
	}

	sanitized := url.Values{}
	for key, vals := range values {
		if s.isSensitiveField(key) {
			sanitized[key] = []string{s.config.Mask}
		} else {
			newVals := make([]string, len(vals))
			for i, val := range vals {
				newVals[i] = s.sanitizeText(val)
			}
			sanitized[key] = newVals
		}
	}

	return sanitized.Encode()
}

// sanitizeText применяет детекторы без regex
func (s *SanitizerNoRegex) sanitizeText(text string) string {
	result := text

	if s.config.EnableBearerTokenDetection {
		result = s.hideBearerTokens(result)
	}

	if s.config.EnableAPIKeyDetection {
		result = s.hideAPIKeys(result)
	}

	if s.config.EnableJWTDetection {
		result = s.hideJWTTokens(result)
	}

	if s.config.EnableCreditCardDetection {
		result = s.hideCreditCards(result)
	}

	if s.config.EnableAWSKeyDetection {
		result = s.hideAWSKeys(result)
	}

	return result
}

// hideBearerTokens скрывает Bearer токены
func (s *SanitizerNoRegex) hideBearerTokens(text string) string {
	result := text
	lower := strings.ToLower(text)

	// Ищем "bearer " (case insensitive)
	idx := 0
	for {
		pos := strings.Index(lower[idx:], "bearer ")
		if pos == -1 {
			break
		}

		pos += idx
		tokenStart := pos + 7 // len("bearer ")

		// Находим конец токена (до пробела или конца строки)
		tokenEnd := tokenStart
		for tokenEnd < len(text) && !isWhitespace(text[tokenEnd]) {
			tokenEnd++
		}

		if tokenEnd > tokenStart {
			// Заменяем токен
			result = result[:tokenStart] + s.config.Mask + result[tokenEnd:]
			lower = strings.ToLower(result)
		}

		idx = pos + 7
		if idx >= len(lower) {
			break
		}
	}

	return result
}

// hideAPIKeys скрывает API ключи
func (s *SanitizerNoRegex) hideAPIKeys(text string) string {
	result := text
	lower := strings.ToLower(text)

	// Паттерны: api_key:, apikey=, api-key:, "api_key":
	patterns := []string{"api_key:", "apikey=", "api-key:", "api_key=", `"api_key":`, `'api_key':`}

	for _, pattern := range patterns {
		idx := 0
		for {
			pos := strings.Index(lower[idx:], pattern)
			if pos == -1 {
				break
			}

			pos += idx
			valueStart := pos + len(pattern)

			// Пропускаем пробелы и кавычки
			for valueStart < len(text) && (isWhitespace(text[valueStart]) || text[valueStart] == '"' || text[valueStart] == '\'') {
				valueStart++
			}

			// Находим конец значения
			valueEnd := valueStart
			for valueEnd < len(text) {
				ch := text[valueEnd]
				if isWhitespace(ch) || ch == '"' || ch == '\'' || ch == ',' || ch == '}' || ch == '&' {
					break
				}
				valueEnd++
			}

			if valueEnd > valueStart && (valueEnd-valueStart) > 10 { // Минимум 10 символов для API ключа
				result = result[:valueStart] + s.config.Mask + result[valueEnd:]
				lower = strings.ToLower(result)
			}

			idx = pos + len(pattern)
			if idx >= len(lower) {
				break
			}
		}
	}

	return result
}

// hideJWTTokens скрывает JWT токены (eyJ...)
func (s *SanitizerNoRegex) hideJWTTokens(text string) string {
	result := text
	idx := 0

	for {
		// JWT всегда начинается с eyJ
		pos := strings.Index(result[idx:], "eyJ")
		if pos == -1 {
			break
		}

		pos += idx
		tokenEnd := pos + 3

		// JWT состоит из base64 символов и точек
		dotCount := 0
		for tokenEnd < len(result) {
			ch := result[tokenEnd]
			if isBase64Char(ch) || ch == '.' {
				if ch == '.' {
					dotCount++
				}
				tokenEnd++
			} else {
				break
			}
		}

		// JWT имеет 2 точки (3 части)
		if dotCount == 2 && (tokenEnd-pos) > 50 {
			result = result[:pos] + s.config.Mask + result[tokenEnd:]
		}

		idx = pos + 3
		if idx >= len(result) {
			break
		}
	}

	return result
}

// hideCreditCards скрывает номера кредитных карт
func (s *SanitizerNoRegex) hideCreditCards(text string) string {
	result := text

	// Удаляем все не-цифры для проверки
	digits := extractDigits(text)

	// Ищем последовательности 13-19 цифр
	for i := 0; i < len(digits)-12; i++ {
		for length := 13; length <= 19 && i+length <= len(digits); length++ {
			cardNum := digits[i : i+length]
			if s.looksLikeCreditCard(cardNum) {
				// Находим эту последовательность в оригинальном тексте и заменяем
				result = s.replaceCreditCardInText(result, cardNum)
			}
		}
	}

	return result
}

// hideAWSKeys скрывает AWS ключи (AKIA...)
func (s *SanitizerNoRegex) hideAWSKeys(text string) string {
	result := text
	idx := 0

	for {
		pos := strings.Index(result[idx:], "AKIA")
		if pos == -1 {
			break
		}

		pos += idx
		keyEnd := pos + 4

		// AWS access key - 20 символов, только uppercase буквы и цифры
		for keyEnd < len(result) && keyEnd-pos < 20 {
			ch := result[keyEnd]
			if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
				keyEnd++
			} else {
				break
			}
		}

		if keyEnd-pos == 20 {
			result = result[:pos] + s.config.Mask + result[keyEnd:]
		}

		idx = pos + 4
		if idx >= len(result) {
			break
		}
	}

	return result
}

// Вспомогательные функции

func (s *SanitizerNoRegex) isSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, sensitive := range s.config.SensitiveFields {
		if strings.Contains(lower, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

func (s *SanitizerNoRegex) truncateBody(body []byte, contentType string) string {
	maxSize := s.config.MaxBodySize
	if len(body) <= maxSize {
		return s.SanitizeBody(body, contentType)
	}

	truncated := body[:maxSize]
	return string(truncated) + "\n... [truncated, total: " + formatSize(len(body)) + "]"
}

func (s *SanitizerNoRegex) summarizeBody(body []byte, contentType string, size int) string {
	summary := "[Large body - " + formatSize(size) + "]"

	if isJSON(contentType) {
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			switch v := data.(type) {
			case map[string]interface{}:
				summary += " Object with " + fmt.Sprint(len(v)) + " keys"
			case []interface{}:
				summary += " Array with " + fmt.Sprint(len(v)) + " items"
			}
		}
	}

	return summary
}

func (s *SanitizerNoRegex) looksLikeCreditCard(digits string) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	// Простая проверка префиксов (Visa, MasterCard, Amex)
	if strings.HasPrefix(digits, "4") || // Visa
		strings.HasPrefix(digits, "5") || // MasterCard
		strings.HasPrefix(digits, "3") { // Amex
		return true
	}

	return false
}

func (s *SanitizerNoRegex) replaceCreditCardInText(text, cardDigits string) string {
	// Ищем эту последовательность цифр в тексте (может быть с разделителями)
	result := text

	// Пробуем разные варианты форматирования
	patterns := []string{
		cardDigits,
		formatCard(cardDigits, "-"),
		formatCard(cardDigits, " "),
	}

	for _, pattern := range patterns {
		if strings.Contains(result, pattern) {
			result = strings.ReplaceAll(result, pattern, s.config.Mask)
		}
	}

	return result
}

// Утилиты

func indexCaseInsensitive(text, substr string) int {
	lower := strings.ToLower(text)
	lowerSubstr := strings.ToLower(substr)
	return strings.Index(lower, lowerSubstr)
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isBase64Char(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '+' || ch == '/' || ch == '=' || ch == '_' || ch == '-'
}

func extractDigits(text string) string {
	var result strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			result.WriteByte(text[i])
		}
	}
	return result.String()
}

func formatCard(digits, separator string) string {
	if len(digits) != 16 {
		return digits
	}
	return digits[0:4] + separator + digits[4:8] + separator + digits[8:12] + separator + digits[12:16]
}
