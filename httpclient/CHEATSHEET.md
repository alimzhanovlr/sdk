# HTTP Client Logger - Шпаргалка

## 🚀 Быстрый старт (30 секунд)

```go
import "github.com/alimzhanovlr/sdk/httpclient""

// 1. Создаем логгер
logger := httpclient.NewSimpleLogger(httpclient.INFO)

// 2. Создаем клиент
config := httpclient.DefaultLoggingConfig(logger)
rt := httpclient.NewLoggingRoundTripper(http.DefaultTransport, config)
client := &http.Client{Transport: rt}

// 3. Используем
client.Do(req) // Логи автоматически!
```

## 📝 Поддерживаемые форматы

| Формат | Content-Type | Санитизация |
|--------|--------------|-------------|
| JSON | `application/json` | ✅ Объекты и массивы |
| XML | `application/xml` | ✅ Теги и атрибуты |
| Form | `application/x-www-form-urlencoded` | ✅ Key-value пары |
| Multipart | `multipart/form-data` | ✅ Form fields |
| Text | `text/plain` | ✅ Regex паттерны |

## 🔒 Что скрывается автоматически

**Поля:** password, token, secret, api_key, ssn, credit_card, cvv, private_key

**Паттерны:** Bearer tokens, AWS keys, JWT, Credit cards, API keys

**Заголовки:** Authorization, Cookie, X-API-Key

## ⚙️ Основные настройки

### Минимальная конфигурация
```go
config := &httpclient.LoggingConfig{
    Logger:         logger,
    LogRequestBody: true,
}
```

### Production конфигурация
```go
config := &httpclient.LoggingConfig{
    Logger:          logger,
    LogRequestBody:  false,  // ❗ Отключено
    LogResponseBody: false,  // ❗ Отключено
    
    ShouldLog: func(req *http.Request) bool {
        return !strings.HasSuffix(req.URL.Path, "/health")
    },
}
```

### Кастомные чувствительные поля
```go
config := &httpclient.SanitizerConfig{
    SensitiveFields: []string{
        "password", "token",
        "my_secret_field",  // ← Ваше поле
    },
    Mask: "***HIDDEN***",
}
```

## 🎯 Типичные сценарии

### Не логировать файлы
```go
config.ShouldLogBody = func(req *http.Request, ct string, size int) bool {
    return !strings.HasPrefix(ct, "image/") && size < 1024*1024
}
```

### Не логировать эндпоинт
```go
config.ShouldLog = func(req *http.Request) bool {
    return !strings.Contains(req.URL.Path, "/upload")
}
```

### Обрезать большие тела
```go
config := &httpclient.SanitizerConfig{
    MaxBodySize: 10 * 1024, // 10KB
}
```

### Пропустить base64
```go
config.BodyRules = []httpclient.BodyProcessingRule{
    {
        Condition: func(ct string, body []byte, size int) bool {
            return looksLikeBase64(body)
        },
        Action:  httpclient.BodyActionSkip,
        Message: "[Base64 - skipped]",
    },
}
```

## 🔧 Интеграция с логгерами

### Zap
```go
type ZapAdapter struct { logger *zap.Logger }
func (z *ZapAdapter) Info(msg string, fields ...interface{}) {
    z.logger.Info(msg, convertToZapFields(fields)...)
}
```

### Logrus
```go
type LogrusAdapter struct { logger *logrus.Logger }
func (l *LogrusAdapter) Info(msg string, fields ...interface{}) {
    l.logger.WithFields(convertToLogrusFields(fields)).Info(msg)
}
```

## 📊 Примеры вывода

### JSON запрос
```
[2024-01-15 10:30:45] INFO: → HTTP Request
method=POST url=https://api.example.com/users
body={
  "email": "user@example.com",
  "password": "***REDACTED***"
}
```

### Большое тело
```
[2024-01-15 10:30:45] INFO: → HTTP Request
body=[Large body - 2.5 MB] Array with 1000 items
```

### Файл
```
[2024-01-15 10:30:45] INFO: → HTTP Request
body=[Binary content - not logged]
```

## 🚨 Частые ошибки

❌ **Забыли скопировать в outputs**
```go
// Неправильно - логи не будут видны
client.Do(req)

// Правильно - используйте RoundTripper
rt := httpclient.NewLoggingRoundTripper(...)
client := &http.Client{Transport: rt}
```

❌ **Логи слишком большие**
```go
// Неправильно
config.MaxBodySize = 10 * 1024 * 1024 // 10MB!

// Правильно
config.MaxBodySize = 10 * 1024 // 10KB
```

❌ **Чувствительные данные в логах**
```go
// Добавьте в SensitiveFields
config.SensitiveFields = append(config.SensitiveFields, "your_field")
```

## 🎓 Лучшие практики

✅ В dev: `LogLevel = DEBUG`, логировать все  
✅ В prod: `LogLevel = ERROR`, минимум логов  
✅ Всегда проверяйте размер body перед логированием  
✅ Используйте `ShouldLog` для фильтрации  
✅ Регулярно аудируйте логи на утечки  
✅ Настройте rotation для логов

## 📚 Полезные файлы

- `README_v2.md` - Полная документация
- `TIPS_AND_PATTERNS.md` - Продвинутые техники
- `sanitizer_v2.go` - Основной код санитайзера
- `roundtripper_v2.go` - HTTP RoundTripper
- `examples_comprehensive.go` - Примеры всех форматов
- `real_world_examples.go` - Реальные сценарии

## 🆘 Помощь

**Проблема:** Логи не появляются  
**Решение:** Проверьте LogLevel и ShouldLog

**Проблема:** Чувствительные данные логируются  
**Решение:** Добавьте поле в SensitiveFields или паттерн

**Проблема:** Производительность упала  
**Решение:** Отключите LogBody или используйте sampling

## 📞 Контакты

GitHub Issues: [ссылка]  
Документация: `README_v2.md`