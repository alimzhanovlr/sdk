# Microkit Framework

Мощный SDK для создания микросервисов на Go с чистой архитектурой.

## 🚀 Возможности

- **Fiber Web Framework** - Быстрый HTTP сервер
- **Uber FX** - Dependency Injection
- **Структурированное логирование** - Zap logger
- **Распределённая трассировка** - OpenTelemetry + Jaeger
- **Интернационализация (i18n)** - Поддержка множества языков
- **CLI для генерации кода** - Быстрое создание компонентов
- **Чистая архитектура** - Слоистая структура проекта
- **Middleware** - Готовые middleware для трассировки, логирования, i18n

## 📦 Установка

### Установка CLI

```bash
go install github.com/yourorg/microkit/cmd/microkit-cli@latest
```

### Добавление SDK в проект

```bash
go get github.com/yourorg/microkit
```

## 🎯 Быстрый старт

### 1. Создание нового проекта

```bash
# Создать новый микросервис
microkit init my-service

# С кастомным module path
microkit init my-service --module github.com/myorg/my-service
```

Это создаст следующую структуру:

```
my-service/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа приложения
├── internal/
│   ├── domain/                  # Доменный слой
│   │   ├── entity/              # Бизнес-сущности
│   │   └── repository/          # Интерфейсы репозиториев
│   ├── usecase/                 # Слой бизнес-логики
│   ├── delivery/                # Слой доставки
│   │   └── http/                # HTTP handlers
│   └── infrastructure/          # Инфраструктурный слой
│       └── repository/          # Реализации репозиториев
├── config/
│   └── config.yaml              # Конфигурация
├── locales/                     # i18n переводы
│   ├── en.yaml
│   └── ru.yaml
├── migrations/                  # Миграции БД
├── go.mod
├── Makefile
└── README.md
```

### 2. Запуск проекта

```bash
cd my-service
go mod tidy
make run
```

Сервис будет доступен на `http://localhost:8080`

### 3. Генерация компонентов

#### Создание Entity (Сущности)

```bash
microkit generate entity user
```

Создаст `internal/domain/entity/user.go`:

```go
package entity

import "time"

type User struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    // Ваши поля
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (e *User) Validate() error {
    // Валидация
    return nil
}
```

#### Создание Repository

```bash
microkit generate repository user
```

Создаст:
- `internal/domain/repository/user.go` - интерфейс
- `internal/infrastructure/repository/user.go` - реализация

```go
// Интерфейс
type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    GetByID(ctx context.Context, id string) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, limit, offset int) ([]*entity.User, error)
}
```

#### Создание Use Case

```bash
microkit generate usecase create-user
```

Создаст `internal/usecase/create_user.go`:

```go
package usecase

type CreateUserUsecase struct {
    logger   *logger.Logger
    tracer   *tracing.Tracer
    userRepo repository.UserRepository
}

func (u *CreateUserUsecase) Execute(ctx context.Context, input CreateUserInput) error {
    ctx, span := u.tracer.Start(ctx, "CreateUserUsecase.Execute")
    defer span.End()
    
    // Бизнес-логика
    
    return nil
}
```

#### Создание HTTP Handler

```bash
microkit generate handler user
```

Создаст `internal/delivery/http/user.go` с CRUD endpoints:

```go
type UserHandler struct {
    logger *logger.Logger
    // usecases
}

func (h *UserHandler) RegisterRoutes(router fiber.Router) {
    group := router.Group("/user")
    
    group.Get("/", h.List)
    group.Get("/:id", h.Get)
    group.Post("/", h.Create)
    group.Put("/:id", h.Update)
    group.Delete("/:id", h.Delete)
}
```

## 🏗 Архитектура

### Принципы чистой архитектуры

1. **Domain Layer** (внутренний круг)
    - Бизнес-сущности (Entity)
    - Интерфейсы репозиториев
    - Не зависит от внешних слоёв

2. **Use Case Layer**
    - Бизнес-логика приложения
    - Оркестрирует работу репозиториев
    - Не зависит от доставки или инфраструктуры

3. **Delivery Layer**
    - HTTP handlers
    - Валидация входных данных
    - ПреобразованиеDTO

4. **Infrastructure Layer**
    - Реализации репозиториев
    - Работа с БД, внешними API
    - Технические детали

### Поток данных

```
HTTP Request 
    → Handler (Delivery)
    → Use Case (Business Logic)
    → Repository Interface (Domain)
    → Repository Implementation (Infrastructure)
    → Database
```

## 📝 Полный пример: User Service

### 1. Создаём проект

```bash
microkit init user-service
cd user-service
```

### 2. Генерируем компоненты

```bash
# Entity
microkit generate entity user

# Repository
microkit generate repository user

# Use Cases
microkit generate usecase create-user
microkit generate usecase get-user
microkit generate usecase list-users

# Handler
microkit generate handler user
```

### 3. Реализуем Entity

Редактируем `internal/domain/entity/user.go`:

```go
package entity

import (
    "time"
    "github.com/yourorg/microkit/pkg/errors"
)

type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) Validate() error {
    if u.Name == "" {
        return errors.ErrValidation.WithDetails(map[string]interface{}{
            "field": "name",
            "error": "name is required",
        })
    }
    
    if u.Email == "" {
        return errors.ErrValidation.WithDetails(map[string]interface{}{
            "field": "email",
            "error": "email is required",
        })
    }
    
    return nil
}
```

### 4. Реализуем Use Case

Редактируем `internal/usecase/create_user.go`:

```go
package usecase

import (
    "context"
    "time"
    
    "github.com/google/uuid"
    "github.com/yourorg/microkit/pkg/logger"
    "github.com/yourorg/microkit/pkg/tracing"
    
    "user-service/internal/domain/entity"
    "user-service/internal/domain/repository"
)

type CreateUserInput struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

type CreateUserUsecase struct {
    logger   *logger.Logger
    tracer   *tracing.Tracer
    userRepo repository.UserRepository
}

func NewCreateUserUsecase(
    logger *logger.Logger,
    tracer *tracing.Tracer,
    userRepo repository.UserRepository,
) *CreateUserUsecase {
    return &CreateUserUsecase{
        logger:   logger,
        tracer:   tracer,
        userRepo: userRepo,
    }
}

func (u *CreateUserUsecase) Execute(ctx context.Context, input CreateUserInput) (*entity.User, error) {
    ctx, span := u.tracer.Start(ctx, "CreateUserUsecase.Execute")
    defer span.End()
    
    u.logger.Info("Creating user", logger.String("email", input.Email))
    
    // Создаём entity
    user := &entity.User{
        ID:        uuid.New().String(),
        Name:      input.Name,
        Email:     input.Email,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Валидация
    if err := user.Validate(); err != nil {
        return nil, err
    }
    
    // Сохраняем
    if err := u.userRepo.Create(ctx, user); err != nil {
        u.logger.Error("Failed to create user", logger.Error(err))
        return nil, err
    }
    
    u.logger.Info("User created successfully", logger.String("user_id", user.ID))
    
    return user, nil
}
```

### 5. Реализуем Handler

Редактируем `internal/delivery/http/user.go`:

```go
package http

import (
    "github.com/gofiber/fiber/v2"
    
    "github.com/yourorg/microkit/pkg/logger"
    "github.com/yourorg/microkit/pkg/errors"
    "github.com/yourorg/microkit/pkg/middleware"
    
    "user-service/internal/usecase"
)

type UserHandler struct {
    logger         *logger.Logger
    createUserUC   *usecase.CreateUserUsecase
    getUserUC      *usecase.GetUserUsecase
    listUsersUC    *usecase.ListUsersUsecase
}

func NewUserHandler(
    logger *logger.Logger,
    createUserUC *usecase.CreateUserUsecase,
    getUserUC *usecase.GetUserUsecase,
    listUsersUC *usecase.ListUsersUsecase,
) *UserHandler {
    return &UserHandler{
        logger:       logger,
        createUserUC: createUserUC,
        getUserUC:    getUserUC,
        listUsersUC:  listUsersUC,
    }
}

func (h *UserHandler) RegisterRoutes(router fiber.Router) {
    group := router.Group("/users")
    
    group.Get("/", h.List)
    group.Get("/:id", h.Get)
    group.Post("/", h.Create)
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
    ctx := c.UserContext()
    
    var input usecase.CreateUserInput
    if err := c.BodyParser(&input); err != nil {
        return errors.ErrBadRequest
    }
    
    user, err := h.createUserUC.Execute(ctx, input)
    if err != nil {
        return err
    }
    
    return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *UserHandler) Get(c *fiber.Ctx) error {
    ctx := c.UserContext()
    id := c.Params("id")
    
    user, err := h.getUserUC.Execute(ctx, id)
    if err != nil {
        return err
    }
    
    return c.JSON(user)
}

func (h *UserHandler) List(c *fiber.Ctx) error {
    ctx := c.UserContext()
    
    users, err := h.listUsersUC.Execute(ctx)
    if err != nil {
        return err
    }
    
    return c.JSON(fiber.Map{
        "data": users,
    })
}
```

### 6. Подключаем в main.go

```go
package main

import (
    "context"
    "log"

    "go.uber.org/fx"
    
    "github.com/yourorg/microkit/pkg/config"
    "github.com/yourorg/microkit/pkg/logger"
    "github.com/yourorg/microkit/pkg/server"
    "github.com/yourorg/microkit/pkg/tracing"
    "github.com/yourorg/microkit/pkg/i18n"
    "github.com/yourorg/microkit/pkg/middleware"
    
    "user-service/internal/delivery/http"
    "user-service/internal/usecase"
    "user-service/internal/infrastructure/repository"
)

func main() {
    app := fx.New(
        // Modules
        fx.Provide(
            provideConfig,
            logger.New,
            tracing.New,
            i18n.New,
            server.New,
            
            // Repositories
            repository.NewUserRepository,
            
            // Use Cases
            usecase.NewCreateUserUsecase,
            usecase.NewGetUserUsecase,
            usecase.NewListUsersUsecase,
            
            // Handlers
            http.NewUserHandler,
        ),
        
        // Lifecycle
        fx.Invoke(
            setupServer,
            registerRoutes,
        ),
    )

    if err := app.Start(context.Background()); err != nil {
        log.Fatal(err)
    }

    <-app.Done()
}

func provideConfig() (*config.Config, error) {
    return config.Load("config/config.yaml")
}

func setupServer(
    lc fx.Lifecycle,
    srv *server.Server,
    log *logger.Logger,
    tracer *tracing.Tracer,
    i18n *i18n.I18n,
) {
    app := srv.App()
    
    // Middleware
    app.Use(middleware.TracingMiddleware(tracer))
    app.Use(middleware.LoggerMiddleware(log))
    app.Use(middleware.I18nMiddleware(i18n))
    
    // Health check
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"status": "ok"})
    })
    
    srv.Start(lc)
    
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            return tracer.Shutdown(ctx)
        },
    })
}

func registerRoutes(
    srv *server.Server,
    userHandler *http.UserHandler,
) {
    api := srv.App().Group("/api/v1")
    
    userHandler.RegisterRoutes(api)
}
```

## ⚙️ Конфигурация

### config.yaml

```yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30
  write_timeout: 30

logger:
  level: info          # debug, info, warn, error
  format: json         # json, console
  output_path: stdout  # stdout или путь к файлу

tracing:
  enabled: true
  service_name: user-service
  endpoint: http://localhost:14268/api/traces
  sample_rate: 1.0     # 0.0 - 1.0

i18n:
  default_language: en
  supported_languages:
    - en
    - ru
  path: ./locales
```

### Переменные окружения

```bash
# Переопределяют значения из config.yaml
export APP_SERVER_PORT=3000
export APP_LOGGER_LEVEL=debug
export APP_TRACING_ENABLED=true
```

## 🌍 Интернационализация

### 1. Создайте файлы переводов

`locales/en.yaml`:
```yaml
user:
  created: "User created successfully"
  not_found: "User not found"
  
validation:
  required: "{{.Field}} is required"
  invalid_email: "Invalid email format"
```

`locales/ru.yaml`:
```yaml
user:
  created: "Пользователь успешно создан"
  not_found: "Пользователь не найден"
  
validation:
  required: "Поле {{.Field}} обязательно"
  invalid_email: "Неверный формат email"
```

### 2. Используйте в коде

```go
func (h *UserHandler) Create(c *fiber.Ctx) error {
    lang := middleware.GetLanguage(c)
    
    // В use case
    message := h.i18n.T(lang, "user.created", nil)
    
    // С параметрами
    message := h.i18n.T(lang, "validation.required", map[string]interface{}{
        "Field": "email",
    })
}
```

### 3. Запрос с языком

```bash
# Через header
curl -H "Accept-Language: ru" http://localhost:8080/api/v1/users

# Через query parameter
curl http://localhost:8080/api/v1/users?lang=ru
```

## 🔍 Трассировка (Tracing)

### Запуск Jaeger (для разработки)

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

UI доступен на: http://localhost:16686

### Использование в коде

```go
func (u *CreateUserUsecase) Execute(ctx context.Context, input Input) error {
    // Создать span
    ctx, span := u.tracer.Start(ctx, "CreateUserUsecase.Execute")
    defer span.End()
    
    // Добавить атрибуты
    u.tracer.SetAttributes(ctx,
        attribute.String("user.email", input.Email),
        attribute.Int("user.age", input.Age),
    )
    
    // Записать событие
    u.tracer.AddEvent(ctx, "Validating user")
    
    // Записать ошибку
    if err != nil {
        u.tracer.RecordError(ctx, err)
        return err
    }
    
    return nil
}
```

## 📊 Логирование

### Структурированное логирование

```go
func (u *CreateUserUsecase) Execute(ctx context.Context, input Input) error {
    u.logger.Info("Creating user",
        logger.String("email", input.Email),
        logger.String("name", input.Name),
    )
    
    if err != nil {
        u.logger.Error("Failed to create user",
            logger.Error(err),
            logger.String("email", input.Email),
        )
        return err
    }
    
    u.logger.Info("User created successfully",
        logger.String("user_id", user.ID),
    )
    
    return nil
}
```

### Контекстный логгер

```go
// Добавить trace ID
logger := u.logger.WithTraceID(tracing.GetTraceID(ctx))

// Добавить произвольные поля
logger := u.logger.WithFields(
    zap.String("service", "user-service"),
    zap.String("environment", "production"),
)

logger.Info("Processing request")
```

## 🛡 Обработка ошибок

### Использование встроенных ошибок

```go
import "github.com/yourorg/microkit/pkg/errors"

// Стандартные ошибки
return errors.ErrNotFound
return errors.ErrBadRequest
return errors.ErrUnauthorized
return errors.ErrInternal

// С деталями
return errors.ErrValidation.WithDetails(map[string]interface{}{
    "field": "email",
    "error": "invalid format",
})

// Обёртка существующей ошибки
return errors.Wrap(err, "database_error", "Failed to query database", 500)
```

### Создание собственных ошибок

```go
var (
    ErrUserExists = errors.New(
        "user_exists",
        "User already exists",
        http.StatusConflict,
    )
    
    ErrInvalidPassword = errors.New(
        "invalid_password",
        "Password does not meet requirements",
        http.StatusBadRequest,
    )
)
```

## 🧪 Тестирование

### Unit тесты

```go
func TestCreateUserUsecase(t *testing.T) {
    // Setup
    logger, _ := logger.New(logger.Config{Level: "debug", Format: "console"})
    tracer, _ := tracing.New(tracing.Config{Enabled: false})
    
    mockRepo := &MockUserRepository{}
    uc := usecase.NewCreateUserUsecase(logger, tracer, mockRepo)
    
    // Test
    input := usecase.CreateUserInput{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    user, err := uc.Execute(context.Background(), input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "John Doe", user.Name)
}
```

## 📚 Best Practices

### 1. Dependency Injection с FX

Всегда используйте fx.Provide для регистрации зависимостей:

```go
fx.Provide(
    // Инфраструктура
    provideDatabase,
    repository.NewUserRepository,
    
    // Use Cases
    usecase.NewCreateUserUsecase,
    
    // Handlers
    http.NewUserHandler,
)
```

### 2. Контекст в каждом методе

Всегда передавайте context для:
- Отмены операций
- Трассировки
- Таймаутов

```go
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    // Используйте ctx для DB queries
    return r.db.WithContext(ctx).Create(user).Error
}
```

### 3. Логирование и трассировка

В каждом use case:

```go
func (u *Usecase) Execute(ctx context.Context) error {
    ctx, span := u.tracer.Start(ctx, "Usecase.Execute")
    defer span.End()
    
    u.logger.Info("Starting operation")
    
    // ... бизнес-логика
    
    u.logger.Info("Operation completed")
    return nil
}
```

### 4. Валидация на границах

Валидируйте входные данные в handlers, не в use cases:

```go
// Handler
func (h *Handler) Create(c *fiber.Ctx) error {
    var input CreateInput
    if err := c.BodyParser(&input); err != nil {
        return errors.ErrBadRequest
    }
    
    // Валидация
    if err := validate.Struct(input); err != nil {
        return errors.ErrValidation
    }
    
    return h.usecase.Execute(ctx, input)
}
```

### 5. Не экспортируйте implementation details

```go
// ❌ Плохо
type userRepository struct { ... }
func NewUserRepository() *userRepository

// ✅ Хорошо
type userRepository struct { ... }
func NewUserRepository() repository.UserRepository  // интерфейс
```

## 🔧 Расширение SDK

### Создание собственного middleware

```go
package middleware

import "github.com/gofiber/fiber/v2"

func CustomMiddleware(config CustomConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // До обработки запроса
        
        err := c.Next()
        
        // После обработки запроса
        
        return err
    }
}
```

### Добавление новых модулей

```go
// pkg/cache/cache.go
package cache

type Cache struct {
    // ...
}

func New(cfg Config) (*Cache, error) {
    // ...
}

// В main.go
fx.Provide(
    cache.New,
)
```

## 🚀 Деплой

### Docker

```bash
docker build -t user-service .
docker run -p 8080:8080 user-service
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
    spec:
      containers:
      - name: user-service
        image: user-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: APP_SERVER_PORT
          value: "8080"
        - name: APP_LOGGER_LEVEL
          value: "info"
```

## 📖 Дополнительные ресурсы

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Fiber Documentation](https://docs.gofiber.io/)
- [Uber FX](https://uber-go.github.io/fx/)
- [OpenTelemetry](https://opentelemetry.io/)

## 📝 Лицензия

MIT

## 🤝 Вклад

Приветствуются pull requests! Для крупных изменений сначала откройте issue.

---

**Happy coding! 🎉**