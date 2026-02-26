# Практические советы и паттерны

## 🎯 Частые сценарии

### 1. Разные конфигурации для разных окружений

```go
func CreateHTTPClient(env string) *http.Client {
    var logger httpclient.Logger
    var config *httpclient.LoggingConfig
    
    switch env {
    case "production":
        logger = httpclient.NewSimpleLogger(httpclient.ERROR)
        config = &httpclient.LoggingConfig{
            Logger:          logger,
            LogRequestBody:  false, // Не логируем body в проде
            LogResponseBody: false,
            LogHeaders:      true,
            Verbose:         false,
            
            ShouldLog: func(req *http.Request) bool {
                // Логируем только ошибки
                return false
            },
        }
        
    case "staging":
        logger = httpclient.NewSimpleLogger(httpclient.INFO)
        config = &httpclient.LoggingConfig{
            Logger:          logger,
            LogRequestBody:  true,
            LogResponseBody: true,
            LogHeaders:      true,
            
            ShouldLogBody: func(req *http.Request, ct string, size int) bool {
                // Не логируем файлы и большие тела
                return !isBinaryContent(ct) && size < 1*1024*1024
            },
        }
        
    default: // development
        logger = httpclient.NewSimpleLogger(httpclient.DEBUG)
        config = httpclient.DefaultLoggingConfig(logger)
        config.Verbose = true
    }
    
    rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
    
    return &http.Client{
        Transport: rt,
        Timeout:   30 * time.Second,
    }
}
```

### 2. Логирование только для определенных эндпоинтов

```go
config := &httpclient.LoggingConfig{
    Logger: logger,
    
    ShouldLog: func(req *http.Request) bool {
        // Логируем только API эндпоинты
        return strings.HasPrefix(req.URL.Path, "/api/")
    },
    
    ShouldLogBody: func(req *http.Request, ct string, size int) bool {
        // Детальное логирование только для auth эндпоинтов
        if strings.Contains(req.URL.Path, "/auth/") {
            return true
        }
        
        // Для остальных - только небольшие тела
        return size < 10*1024
    },
}
```

### 3. Маскирование PII (Personally Identifiable Information)

```go
config := &httpclient.SanitizerConfig{
    SensitiveFields: []string{
        // Стандартные PII
        "ssn", "social_security_number",
        "passport_number", "driver_license",
        "date_of_birth", "birth_date",
        "tax_id", "national_id",
        
        // Контактные данные
        "email", "phone", "phone_number",
        "mobile", "address", "street_address",
        
        // Финансовые
        "credit_card", "bank_account",
        "iban", "swift", "routing_number",
    },
    
    SensitivePatterns: []*regexp.Regexp{
        // Email
        regexp.MustCompile(`([a-zA-Z0-9._%+-]+@)[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
        
        // Телефон (различные форматы)
        regexp.MustCompile(`\+?[\d\s\-\(\)]{10,}`),
        
        // SSN (xxx-xx-xxxx)
        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
        
        // Credit cards
        regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
    },
    
    Mask: "[PII_REDACTED]",
}
```

### 4. Специфичные правила для медиа контента

```go
config := &httpclient.SanitizerConfig{
    // ... стандартные настройки ...
    
    BodyRules: []httpclient.BodyProcessingRule{
        // Изображения
        {
            Condition: func(ct string, body []byte, size int) bool {
                return strings.HasPrefix(ct, "image/")
            },
            Action:  httpclient.BodyActionSkip,
            Message: fmt.Sprintf("[Image - %s - not logged]", formatSize(size)),
        },
        
        // Видео
        {
            Condition: func(ct string, body []byte, size int) bool {
                return strings.HasPrefix(ct, "video/")
            },
            Action:  httpclient.BodyActionSkip,
            Message: "[Video content - not logged]",
        },
        
        // PDF документы
        {
            Condition: func(ct string, body []byte, size int) bool {
                return ct == "application/pdf"
            },
            Action:  httpclient.BodyActionSummarize,
            Message: fmt.Sprintf("[PDF document - %s]", formatSize(size)),
        },
        
        // Base64 encoded файлы
        {
            Condition: func(ct string, body []byte, size int) bool {
                if !isJSON(ct) {
                    return false
                }
                // Проверяем есть ли в JSON поля с base64
                var data map[string]interface{}
                if json.Unmarshal(body, &data) != nil {
                    return false
                }
                
                for key, val := range data {
                    if strings.Contains(strings.ToLower(key), "image") ||
                       strings.Contains(strings.ToLower(key), "file") ||
                       strings.Contains(strings.ToLower(key), "data") {
                        if str, ok := val.(string); ok && len(str) > 1000 && looksLikeBase64([]byte(str)) {
                            return true
                        }
                    }
                }
                return false
            },
            Action:  httpclient.BodyActionSummarize,
            Message: "[JSON with base64 encoded files]",
        },
    },
}
```

### 5. Умная обработка ответов с пагинацией

```go
config.BodyRules = append(config.BodyRules, httpclient.BodyProcessingRule{
    Condition: func(ct string, body []byte, size int) bool {
        if !isJSON(ct) {
            return false
        }
        
        var data map[string]interface{}
        if json.Unmarshal(body, &data) != nil {
            return false
        }
        
        // Если это список с pagination
        if items, ok := data["items"].([]interface{}); ok && len(items) > 100 {
            return true
        }
        
        if results, ok := data["results"].([]interface{}); ok && len(results) > 100 {
            return true
        }
        
        return false
    },
    Action: httpclient.BodyActionSummarize,
})
```

## 🔧 Продвинутые техники

### 1. Контекстное логирование

```go
type ContextLogger struct {
    logger httpclient.Logger
}

func (c *ContextLogger) Debug(msg string, fields ...interface{}) {
    // Добавляем контекстные поля из окружения
    enriched := append(fields,
        "service", os.Getenv("SERVICE_NAME"),
        "version", os.Getenv("VERSION"),
        "pod", os.Getenv("HOSTNAME"),
    )
    c.logger.Debug(msg, enriched...)
}

func (c *ContextLogger) Info(msg string, fields ...interface{}) {
    enriched := append(fields,
        "service", os.Getenv("SERVICE_NAME"),
        "version", os.Getenv("VERSION"),
    )
    c.logger.Info(msg, enriched...)
}

func (c *ContextLogger) Error(msg string, fields ...interface{}) {
    enriched := append(fields,
        "service", os.Getenv("SERVICE_NAME"),
        "version", os.Getenv("VERSION"),
        "environment", os.Getenv("ENV"),
    )
    c.logger.Error(msg, enriched...)
}
```

### 2. Динамическое изменение уровня логов

```go
type DynamicLogger struct {
    baseLogger httpclient.Logger
    level      *atomic.Value // stores LogLevel
}

func NewDynamicLogger(initial httpclient.LogLevel) *DynamicLogger {
    level := &atomic.Value{}
    level.Store(initial)
    
    return &DynamicLogger{
        baseLogger: httpclient.NewSimpleLogger(initial),
        level:      level,
    }
}

func (d *DynamicLogger) SetLevel(level httpclient.LogLevel) {
    d.level.Store(level)
    d.baseLogger = httpclient.NewSimpleLogger(level)
}

func (d *DynamicLogger) Debug(msg string, fields ...interface{}) {
    if d.level.Load().(httpclient.LogLevel) <= httpclient.DEBUG {
        d.baseLogger.Debug(msg, fields...)
    }
}

// HTTP endpoint для изменения уровня логов
func logLevelHandler(logger *DynamicLogger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        level := r.URL.Query().Get("level")
        switch level {
        case "debug":
            logger.SetLevel(httpclient.DEBUG)
        case "info":
            logger.SetLevel(httpclient.INFO)
        case "error":
            logger.SetLevel(httpclient.ERROR)
        default:
            http.Error(w, "Invalid level", http.StatusBadRequest)
            return
        }
        w.WriteHeader(http.StatusOK)
    }
}
```

### 3. Метрики на основе логов

```go
type MetricsRoundTripper struct {
    next    http.RoundTripper
    metrics *Metrics
}

type Metrics struct {
    requestsTotal     prometheus.Counter
    requestDuration   prometheus.Histogram
    requestSizeBytes  prometheus.Histogram
    responseSizeBytes prometheus.Histogram
}

func (m *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    start := time.Now()
    
    // Измеряем размер запроса
    var reqSize int64
    if req.Body != nil {
        body, _ := io.ReadAll(req.Body)
        reqSize = int64(len(body))
        req.Body = io.NopCloser(bytes.NewBuffer(body))
        m.metrics.requestSizeBytes.Observe(float64(reqSize))
    }
    
    resp, err := m.next.RoundTrip(req)
    
    duration := time.Since(start)
    m.metrics.requestsTotal.Inc()
    m.metrics.requestDuration.Observe(duration.Seconds())
    
    if resp != nil && resp.Body != nil {
        body, _ := io.ReadAll(resp.Body)
        m.metrics.responseSizeBytes.Observe(float64(len(body)))
        resp.Body = io.NopCloser(bytes.NewBuffer(body))
    }
    
    return resp, err
}

// Комбинируем с логированием
func CreateMonitoredClient(logger httpclient.Logger, metrics *Metrics) *http.Client {
    base := http.DefaultTransport
    
    // Сначала метрики
    withMetrics := &MetricsRoundTripper{
        next:    base,
        metrics: metrics,
    }
    
    // Потом логирование
    withLogging := httpclient.NewLoggingRoundTripper(
        withMetrics,
        httpclient.DefaultLoggingConfig(logger),
    )
    
    return &http.Client{Transport: withLogging}
}
```

### 4. Structured logging для ELK/Splunk

```go
type StructuredLogger struct {
    encoder *json.Encoder
}

func NewStructuredLogger() *StructuredLogger {
    return &StructuredLogger{
        encoder: json.NewEncoder(os.Stdout),
    }
}

func (s *StructuredLogger) log(level string, msg string, fields ...interface{}) {
    entry := map[string]interface{}{
        "@timestamp": time.Now().UTC().Format(time.RFC3339),
        "level":      level,
        "message":    msg,
    }
    
    // Добавляем поля
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprint(fields[i])
            entry[key] = fields[i+1]
        }
    }
    
    s.encoder.Encode(entry)
}

func (s *StructuredLogger) Debug(msg string, fields ...interface{}) {
    s.log("debug", msg, fields...)
}

func (s *StructuredLogger) Info(msg string, fields ...interface{}) {
    s.log("info", msg, fields...)
}

func (s *StructuredLogger) Error(msg string, fields ...interface{}) {
    s.log("error", msg, fields...)
}
```

## 🚨 Troubleshooting

### Проблема: Логи слишком большие

**Решение:**
```go
config.MaxBodySize = 5 * 1024 // Уменьшить до 5KB
config.LogRequestBody = false // Отключить для запросов
config.ShouldLogBody = func(req *http.Request, ct string, size int) bool {
    // Логировать только маленькие тела
    return size < 1024
}
```

### Проблема: Чувствительные данные все еще логируются

**Решение:**
```go
// Добавьте больше паттернов
config.SensitivePatterns = append(config.SensitivePatterns,
    regexp.MustCompile(`ваш-специфичный-паттерн`),
)

// Или используйте кастомный санитайзер
type CustomSanitizer struct {
    *httpclient.Sanitizer
}

func (c *CustomSanitizer) SanitizeBody(body []byte, ct string) string {
    // Ваша логика
    result := c.Sanitizer.SanitizeBody(body, ct)
    // Дополнительная обработка
    return result
}
```

### Проблема: Производительность страдает

**Решение:**
```go
// Отключите логирование для high-throughput эндпоинтов
config.ShouldLog = func(req *http.Request) bool {
    // Не логируем метрики и health checks
    if strings.HasSuffix(req.URL.Path, "/metrics") ||
       strings.HasSuffix(req.URL.Path, "/health") {
        return false
    }
    return true
}

// Или используйте sampling
var logCounter atomic.Uint64
config.ShouldLog = func(req *http.Request) bool {
    // Логируем каждый 100-й запрос
    return logCounter.Add(1)%100 == 0
}
```

## 📊 Примеры интеграций

### DataDog

```go
type DatadogLogger struct {
    client *statsd.Client
}

func (d *DatadogLogger) Info(msg string, fields ...interface{}) {
    // Отправляем метрики в DataDog
    d.client.Incr("http.requests", []string{"status:success"}, 1)
    
    // И логируем
    log.WithFields(fieldsToMap(fields)).Info(msg)
}
```

### New Relic

```go
type NewRelicLogger struct {
    app newrelic.Application
}

func (n *NewRelicLogger) Info(msg string, fields ...interface{}) {
    // Создаем transaction
    txn := n.app.StartTransaction("http_request")
    defer txn.End()
    
    // Добавляем атрибуты
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            txn.AddAttribute(fmt.Sprint(fields[i]), fields[i+1])
        }
    }
}
```

## 🎓 Советы по безопасности

1. **Никогда не логируйте:**
    - Пароли
    - Токены доступа
    - API ключи
    - Номера кредитных карт
    - SSN и другие PII
    - Private keys

2. **Всегда проверяйте:**
    - Регулярно аудируйте логи
    - Используйте automated scanning для поиска утечек
    - Настройте alerts на появление чувствительных паттернов

3. **В production:**
    - Минимизируйте логирование
    - Используйте encrypted логи
    - Настройте log rotation
    - Ограничьте доступ к логам

4. **Compliance (GDPR, HIPAA, etc):**
    - Документируйте что логируется
    - Настройте retention policies
    - Обеспечьте right to deletion