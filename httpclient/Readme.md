# HTTP Client с продвинутым логированием и санитизацией

Полнофункциональная библиотека для HTTP клиента с автоматическим логированием и санитизацией чувствительных данных во всех популярных форматах.

## 🎯 Возможности

### Поддерживаемые форматы
✅ **JSON** - объекты и массивы  
✅ **XML** - теги и атрибуты  
✅ **Form URL-encoded** - `application/x-www-form-urlencoded`  
✅ **Multipart Form** - `multipart/form-data`  
✅ **Plain Text** - с regex паттернами  
✅ **Headers** - с гибкой санитизацией  
✅ **Query Parameters** - в URL

### Дополнительные фичи
✅ Обработка экранированных JSON строк (`\"`)  
✅ Автоматическое определение формата  
✅ Лимиты на размер логируемого body  
✅ Умная обработка больших тел (truncate, summarize, skip)  
✅ Детектор бинарных данных и base64  
✅ Кастомные правила обработки  
✅ Гибкая санитизация заголовков (full/partial mask)  
✅ Интеграция с любым логгером

## 🚀 Быстрый старт

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
    "your-module/httpclient"
)

func main() {
    // Создаем логгер
    logger := httpclient.NewSimpleLogger(httpclient.INFO)

    // Создаем HTTP клиент с логированием (дефолтные настройки)
    config := httpclient.DefaultLoggingConfig(logger)
    rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
    
    client := &http.Client{
        Transport: rt,
        Timeout:   30 * time.Second,
    }

    // Используем как обычный клиент
    payload := map[string]string{
        "username": "user@example.com",
        "password": "secret123",  // Автоматически скроется в логах!
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", "https://api.example.com/login", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer sk-secret-token") // Тоже скроется!
    
    resp, _ := client.Do(req)
    // ...
}
```

## 📋 Примеры для разных форматов

### JSON (объект и массив)

```go
// JSON объект
payload := map[string]interface{}{
    "username": "user",
    "password": "secret123",
    "api_key":  "sk-1234567890",
}

// Лог вывод:
{
  "username": "user",
  "password": "***REDACTED***",
  "api_key": "***REDACTED***"
}

// JSON массив
arrayPayload := []map[string]interface{}{
    {"id": 1, "token": "tok1"},
    {"id": 2, "token": "tok2"},
}

// Лог вывод:
[
  {"id": 1, "token": "***REDACTED***"},
  {"id": 2, "token": "***REDACTED***"}
]
```

### XML

```go
xmlBody := `<?xml version="1.0"?>
<user>
    <username>john</username>
    <password>secret123</password>
    <api_key>sk-key-xyz</api_key>
</user>`

req, _ := http.NewRequest("POST", url, strings.NewReader(xmlBody))
req.Header.Set("Content-Type", "application/xml")

// Лог вывод:
<user>
    <username>john</username>
    <password>***REDACTED***</password>
    <api_key>***REDACTED***</api_key>
</user>
```

### Form URL-encoded

```go
formData := "username=user&password=secret123&api_key=sk-key"

req, _ := http.NewRequest("POST", url, strings.NewReader(formData))
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// Лог вывод:
username=user&password=***REDACTED***&api_key=***REDACTED***
```

### Multipart Form

```go
multipartBody := `------Boundary
Content-Disposition: form-data; name="username"

john_doe
------Boundary
Content-Disposition: form-data; name="password"

secret123
------Boundary--`

// Лог вывод:
------Boundary
Content-Disposition: form-data; name="username"
john_doe
------Boundary
Content-Disposition: form-data; name="password"
***REDACTED***
------Boundary--
```

### Headers

```go
req.Header.Set("Authorization", "Bearer sk-1234567890abcdefghijklmnop")
req.Header.Set("X-API-Key", "api-key-secret")
req.Header.Set("User-Agent", "MyApp/1.0")

// Лог вывод (HeaderMaskMode = Partial):
headers: {
  "Authorization": "Bear***REDACTED***mnop",
  "X-Api-Key": "***REDACTED***",
  "User-Agent": "MyApp/1.0"
}
```

## ⚙️ Конфигурация

### Базовая конфигурация

```go
config := &httpclient.LoggingConfig{
    Logger:          logger,
    LogRequestBody:  true,  // Логировать body запроса
    LogResponseBody: true,  // Логировать body ответа
    LogHeaders:      true,  // Логировать заголовки
    Verbose:         false, // Детальные логи
    
    SanitizerConfig: nil,   // nil = дефолтные настройки
}
```

### Продвинутая конфигурация

```go
config := &httpclient.SanitizerConfig{
    // Чувствительные поля (case-insensitive)
    SensitiveFields: []string{
        "password", "token", "secret", "api_key",
        "ssn", "credit_card", "private_key",
        // Ваши кастомные поля:
        "internal_key", "webhook_secret",
    },
    
    // Regex паттерны для текста
    SensitivePatterns: []*regexp.Regexp{
        // Bearer tokens
        regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9\-._~+/]+=*`),
        
        // Email (опционально)
        regexp.MustCompile(`([a-zA-Z0-9._%+-]+@)[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
        
        // AWS keys
        regexp.MustCompile(`(AKIA[0-9A-Z]{16})`),
        
        // JWT tokens
        regexp.MustCompile(`(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`),
        
        // Credit cards
        regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14})\b`),
    },
    
    // Маска замены
    Mask: "***REDACTED***",
    
    // Максимальный размер body для логирования
    MaxBodySize: 100 * 1024, // 100KB
    
    // Режим маскирования заголовков
    HeaderMaskMode: httpclient.HeaderMaskPartial, // full или partial
    
    // Дополнительные чувствительные заголовки
    SensitiveHeaders: []string{
        "x-custom-auth", "x-internal-token",
    },
}
```

### Правила обработки больших тел

```go
config := &httpclient.SanitizerConfig{
    // ... другие настройки ...
    
    BodyRules: []httpclient.BodyProcessingRule{
        // Правило 1: Пропускаем бинарные файлы
        {
            Condition: func(contentType string, body []byte, size int) bool {
                return isBinaryContent(contentType)
            },
            Action:  httpclient.BodyActionSkip,
            Message: "[Binary file - not logged]",
        },
        
        // Правило 2: Пропускаем base64 больше 1KB
        {
            Condition: func(contentType string, body []byte, size int) bool {
                return size > 1024 && looksLikeBase64(body)
            },
            Action:  httpclient.BodyActionSkip,
            Message: "[Base64 data - not logged]",
        },
        
        // Правило 3: Суммаризуем огромные JSON (>500KB)
        {
            Condition: func(contentType string, body []byte, size int) bool {
                return size > 500*1024 && isJSON(contentType)
            },
            Action: httpclient.BodyActionSummarize,
            // Message автоматически: "[Large body - 1.2 MB] Array with 5000 items"
        },
        
        // Правило 4: Обрезаем большие тела
        {
            Condition: func(contentType string, body []byte, size int) bool {
                return size > 100*1024
            },
            Action: httpclient.BodyActionTruncate,
        },
    },
}
```

### Кастомные условия логирования

```go
config := &httpclient.LoggingConfig{
    Logger: logger,
    
    // Определяем когда логировать запрос
    ShouldLog: func(req *http.Request) bool {
        // Не логируем health checks
        if req.URL.Path == "/health" {
            return false
        }
        return true
    },
    
    // Определяем когда логировать body
    ShouldLogBody: func(req *http.Request, contentType string, size int) bool {
        // Не логируем body для файловых эндпоинтов
        if strings.Contains(req.URL.Path, "/upload") {
            return false
        }
        
        // Не логируем изображения
        if strings.HasPrefix(contentType, "image/") {
            return false
        }
        
        // Не логируем очень большие тела
        if size > 10*1024*1024 { // 10MB
            return false
        }
        
        return true
    },
}
```

## 🔒 Дефолтные чувствительные поля

По умолчанию санитизируются (case-insensitive):

**Аутентификация:**
- password, passwd, pwd, secret, token
- api_key, apikey, api_secret
- access_token, refresh_token
- client_secret, authorization, auth
- bearer, session, session_id, cookie

**Персональные данные:**
- ssn, social_security, passport
- driver_license, tax_id, ein, vat

**Финансовые:**
- credit_card, card_number, cvv, cvc
- pin, account_number, routing_number
- iban, swift

**Криптография:**
- private_key, public_key, encryption_key
- certificate, cert, key, pem

**Сервисы:**
- stripe_key, aws_secret, gcp_key
- azure_key, webhook_secret

## 🎨 Интеграция с логгерами

### Zap

```go
type ZapAdapter struct {
    logger *zap.Logger
}

func (z *ZapAdapter) Debug(msg string, fields ...interface{}) {
    z.logger.Debug(msg, convertToZapFields(fields)...)
}

func (z *ZapAdapter) Info(msg string, fields ...interface{}) {
    z.logger.Info(msg, convertToZapFields(fields)...)
}

func (z *ZapAdapter) Error(msg string, fields ...interface{}) {
    z.logger.Error(msg, convertToZapFields(fields)...)
}

func convertToZapFields(fields []interface{}) []zap.Field {
    zapFields := make([]zap.Field, 0, len(fields)/2)
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprint(fields[i])
            zapFields = append(zapFields, zap.Any(key, fields[i+1]))
        }
    }
    return zapFields
}
```

### Logrus

```go
type LogrusAdapter struct {
    logger *logrus.Logger
}

func (l *LogrusAdapter) Debug(msg string, fields ...interface{}) {
    l.logger.WithFields(convertToLogrusFields(fields)).Debug(msg)
}

func (l *LogrusAdapter) Info(msg string, fields ...interface{}) {
    l.logger.WithFields(convertToLogrusFields(fields)).Info(msg)
}

func (l *LogrusAdapter) Error(msg string, fields ...interface{}) {
    l.logger.WithFields(convertToLogrusFields(fields)).Error(msg)
}

func convertToLogrusFields(fields []interface{}) logrus.Fields {
    logrusFields := make(logrus.Fields)
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprint(fields[i])
            logrusFields[key] = fields[i+1]
        }
    }
    return logrusFields
}
```

## 📊 Примеры вывода логов

### Успешный запрос

```
[2024-01-15 10:30:45.123] INFO: → HTTP Request | method=POST url=https://api.example.com/users host=api.example.com headers=map[Authorization:Bear***REDACTED***xyz Content-Type:application/json] body={
  "email": "user@example.com",
  "password": "***REDACTED***",
  "api_key": "***REDACTED***"
}

[2024-01-15 10:30:45.456] DEBUG: ← HTTP Response | method=POST url=https://api.example.com/users status=201 status_text=201 Created duration_ms=333 body={
  "id": "user_123",
  "email": "user@example.com",
  "token": "***REDACTED***"
}
```

### Большое тело

```
[2024-01-15 10:31:00.123] INFO: → HTTP Request | method=POST url=https://api.example.com/data body=[Large body - 2.5 MB] Array with 10000 items
```

### Бинарный файл

```
[2024-01-15 10:31:15.456] INFO: → HTTP Request | method=POST url=https://api.example.com/upload headers=map[Content-Type:image/png] body=[Binary content - not logged]
```

### Base64 данные

```
[2024-01-15 10:31:30.789] INFO: → HTTP Request | method=POST url=https://api.example.com/image body=[Base64 encoded data - not logged]
```

## 💡 Best Practices

### 1. Не логируйте всё в продакшене
```go
config := httpclient.DefaultLoggingConfig(logger)
if env == "production" {
    config.LogRequestBody = false
    config.LogResponseBody = false
    config.Verbose = false
}
```

### 2. Используйте правила для оптимизации
```go
// Не логируем файлы и очень большие тела
config.ShouldLogBody = func(req *http.Request, contentType string, size int) bool {
    return !isBinaryContent(contentType) && size < 10*1024*1024
}
```

### 3. Добавляйте свои паттерны
```go
// Для вашего специфичного API
config.SensitivePatterns = append(
    config.SensitivePatterns,
    regexp.MustCompile(`(myapp-key-)[a-zA-Z0-9]{32}`),
)
```

### 4. Настройте уровни логов
```go
// Детальные логи в dev, минимальные в prod
if env == "production" {
    logger = httpclient.NewSimpleLogger(httpclient.ERROR)
} else {
    logger = httpclient.NewSimpleLogger(httpclient.DEBUG)
}
```

### 5. Комбинируйте RoundTripper'ы
```go
// Базовый транспорт
base := http.DefaultTransport

// Добавляем tracing
tracing := NewTracingRoundTripper(base)

// Добавляем rate limiting
rateLimited := NewRateLimitingRoundTripper(tracing, 100)

// Добавляем логирование
logging := httpclient.NewLoggingRoundTripper(rateLimited, config)

client := &http.Client{Transport: logging}
```

## 🧪 Тестирование

```bash
# Запустить все тесты
go test -v

# Запустить конкретный тест
go test -v -run TestSanitizer_JSON

# Проверить покрытие
go test -cover

# Подробное покрытие
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📦 Установка

```bash
go get github.com/yourusername/httpclient
```

## 🤝 Вклад

Issues и Pull Requests приветствуются!

## 📄 Лицензия

MIT License

## 🔗 Полезные ссылки

- [Документация по regex паттернам](https://regex101.com/)
- [HTTP Content-Type список](https://www.iana.org/assignments/media-types/media-types.xhtml)
- [Best practices для логирования](https://12factor.net/logs)