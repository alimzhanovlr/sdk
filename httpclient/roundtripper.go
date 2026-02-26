package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// Logger интерфейс для логирования
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// LoggingRoundTripper RoundTripper с логированием и санитизацией
type LoggingRoundTripper struct {
	next      http.RoundTripper
	logger    Logger
	sanitizer *Sanitizer
	config    *LoggingConfig
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Logger          Logger
	SanitizerConfig *SanitizerConfig

	// Логировать ли тело запроса/ответа
	LogRequestBody  bool
	LogResponseBody bool

	// Логировать ли заголовки
	LogHeaders bool

	// Функция для определения нужно ли логировать конкретный запрос
	ShouldLog func(req *http.Request) bool

	// Функция для определения нужно ли логировать body для конкретного запроса
	ShouldLogBody func(req *http.Request, contentType string, size int) bool

	// Уровень детализации логов
	Verbose bool
}

// DefaultLoggingConfig дефолтная конфигурация
func DefaultLoggingConfig(logger Logger) *LoggingConfig {
	return &LoggingConfig{
		Logger:          logger,
		SanitizerConfig: nil, // Будет использован дефолтный
		LogRequestBody:  true,
		LogResponseBody: true,
		LogHeaders:      true,
		Verbose:         false,

		// По умолчанию логируем все
		ShouldLog: func(req *http.Request) bool {
			return true
		},

		// Пропускаем body для файлов и очень больших запросов
		ShouldLogBody: func(req *http.Request, contentType string, size int) bool {
			// Не логируем файлы
			if isBinaryContent(contentType) {
				return false
			}
			// Не логируем очень большие тела
			if size > 10*1024*1024 { // 10MB
				return false
			}
			return true
		},
	}
}

// NewLoggingRoundTripper создает RoundTripper с логированием
func NewLoggingRoundTripper(next http.RoundTripper, config *LoggingConfig) *LoggingRoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	if config == nil {
		config = &LoggingConfig{
			LogRequestBody:  true,
			LogResponseBody: true,
			LogHeaders:      true,
		}
	}

	sanitizer := NewSanitizer(config.SanitizerConfig)

	return &LoggingRoundTripper{
		next:      next,
		logger:    config.Logger,
		sanitizer: sanitizer,
		config:    config,
	}
}

// RoundTrip выполняет HTTP запрос с логированием
func (l *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Проверяем нужно ли логировать этот запрос
	if l.config.ShouldLog != nil && !l.config.ShouldLog(req) {
		return l.next.RoundTrip(req)
	}

	start := time.Now()

	// Логируем запрос
	l.logRequest(req)

	// Выполняем запрос
	resp, err := l.next.RoundTrip(req)

	duration := time.Since(start)

	// Логируем ответ или ошибку
	if err != nil {
		l.logError(req, err, duration)
		return nil, err
	}

	l.logResponse(req, resp, duration)

	return resp, nil
}

// logRequest логирует исходящий запрос
func (l *LoggingRoundTripper) logRequest(req *http.Request) {
	if l.logger == nil {
		return
	}

	fields := []interface{}{
		"method", req.Method,
		"url", l.sanitizeURL(req.URL),
		"host", req.Host,
	}

	// Добавляем path и query отдельно для удобства
	if l.config.Verbose {
		fields = append(fields, "path", req.URL.Path)
		if req.URL.RawQuery != "" {
			fields = append(fields, "query", l.sanitizeQuery(req.URL.RawQuery))
		}
	}

	// Логируем заголовки
	if l.config.LogHeaders && len(req.Header) > 0 {
		headers := l.sanitizer.SanitizeHeaders(map[string][]string(req.Header))
		fields = append(fields, "headers", headers)
	}

	// Логируем тело
	if l.config.LogRequestBody && req.Body != nil {
		body := l.readAndRestoreBody(&req.Body)
		if len(body) > 0 {
			contentType := req.Header.Get("Content-Type")

			// Проверяем нужно ли логировать body
			shouldLog := true
			if l.config.ShouldLogBody != nil {
				shouldLog = l.config.ShouldLogBody(req, contentType, len(body))
			}

			if shouldLog {
				sanitized := l.sanitizer.SanitizeBody(body, contentType)
				fields = append(fields, "body", sanitized)
			} else {
				fields = append(fields, "body", fmt.Sprintf("[Body not logged - size: %s]", formatSize(len(body))))
			}
		}
	}

	l.logger.Info("→ HTTP Request", fields...)
}

// logResponse логирует ответ
func (l *LoggingRoundTripper) logResponse(req *http.Request, resp *http.Response, duration time.Duration) {
	if l.logger == nil {
		return
	}

	fields := []interface{}{
		"method", req.Method,
		"url", l.sanitizeURL(req.URL),
		"status", resp.StatusCode,
		"status_text", resp.Status,
		"duration_ms", duration.Milliseconds(),
	}

	// Добавляем размер ответа
	if l.config.Verbose && resp.ContentLength > 0 {
		fields = append(fields, "content_length", formatSize(int(resp.ContentLength)))
	}

	// Логируем заголовки
	if l.config.LogHeaders && len(resp.Header) > 0 {
		headers := l.sanitizer.SanitizeHeaders(map[string][]string(resp.Header))
		fields = append(fields, "headers", headers)
	}

	// Логируем тело
	if l.config.LogResponseBody && resp.Body != nil {
		body := l.readAndRestoreBody(&resp.Body)
		if len(body) > 0 {
			contentType := resp.Header.Get("Content-Type")

			// Проверяем нужно ли логировать body
			shouldLog := true
			if l.config.ShouldLogBody != nil {
				shouldLog = l.config.ShouldLogBody(req, contentType, len(body))
			}

			if shouldLog {
				sanitized := l.sanitizer.SanitizeBody(body, contentType)
				fields = append(fields, "body", sanitized)
			} else {
				fields = append(fields, "body", fmt.Sprintf("[Body not logged - size: %s]", formatSize(len(body))))
			}
		}
	}

	// Выбираем уровень лога
	if resp.StatusCode >= 500 {
		l.logger.Error("← HTTP Response", fields...)
	} else if resp.StatusCode >= 400 {
		l.logger.Info("← HTTP Response", fields...)
	} else {
		l.logger.Debug("← HTTP Response", fields...)
	}
}

// logError логирует ошибку
func (l *LoggingRoundTripper) logError(req *http.Request, err error, duration time.Duration) {
	if l.logger == nil {
		return
	}

	l.logger.Error("✗ HTTP Request Failed",
		"method", req.Method,
		"url", l.sanitizeURL(req.URL),
		"error", err.Error(),
		"duration_ms", duration.Milliseconds(),
	)
}

// sanitizeURL санитизирует URL (скрывает чувствительные query параметры)
func (l *LoggingRoundTripper) sanitizeURL(u *url.URL) string {
	if u.RawQuery == "" {
		return u.String()
	}

	sanitizedQuery := l.sanitizeQuery(u.RawQuery)

	result := u.Scheme + "://" + u.Host + u.Path
	if sanitizedQuery != "" {
		result += "?" + sanitizedQuery
	}
	if u.Fragment != "" {
		result += "#" + u.Fragment
	}

	return result
}

// sanitizeQuery санитизирует query параметры
func (l *LoggingRoundTripper) sanitizeQuery(rawQuery string) string {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	sanitized := url.Values{}
	for key, vals := range values {
		if l.sanitizer.isSensitiveField(key) {
			sanitized[key] = []string{l.sanitizer.config.Mask}
		} else {
			sanitized[key] = vals
		}
	}

	return sanitized.Encode()
}

// readAndRestoreBody читает тело и восстанавливает его
func (l *LoggingRoundTripper) readAndRestoreBody(body *io.ReadCloser) []byte {
	if body == nil || *body == nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(*body)
	if err != nil {
		return nil
	}

	// Восстанавливаем для дальнейшего использования
	*body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes
}

// DumpRequest возвращает полный дамп запроса (для отладки)
func (l *LoggingRoundTripper) DumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return fmt.Sprintf("Error dumping request: %v", err)
	}
	return string(dump)
}

// DumpResponse возвращает полный дамп ответа (для отладки)
func (l *LoggingRoundTripper) DumpResponse(resp *http.Response) string {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return fmt.Sprintf("Error dumping response: %v", err)
	}
	return string(dump)
}

// WithLogger создает новый RoundTripper с другим логгером
func (l *LoggingRoundTripper) WithLogger(logger Logger) *LoggingRoundTripper {
	config := *l.config
	config.Logger = logger
	return NewLoggingRoundTripper(l.next, &config)
}

// WithoutBodyLogging отключает логирование body
func (l *LoggingRoundTripper) WithoutBodyLogging() *LoggingRoundTripper {
	config := *l.config
	config.LogRequestBody = false
	config.LogResponseBody = false
	return NewLoggingRoundTripper(l.next, &config)
}
