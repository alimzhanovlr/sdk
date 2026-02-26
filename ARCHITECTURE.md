# Архитектура Microkit Framework

## Обзор

Microkit следует принципам **Clean Architecture** (Чистой Архитектуры) с четким разделением ответственности между слоями.

## Слои архитектуры

```
┌─────────────────────────────────────────────────────┐
│                  Delivery Layer                     │
│            (HTTP, gRPC, CLI, Message Queue)         │
│                                                     │
│   ┌─────────────────────────────────────────┐      │
│   │           Use Case Layer                │      │
│   │       (Business Logic)                  │      │
│   │                                         │      │
│   │   ┌─────────────────────────────┐      │      │
│   │   │      Domain Layer          │      │      │
│   │   │  (Entities, Interfaces)    │      │      │
│   │   └─────────────────────────────┘      │      │
│   └─────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│             Infrastructure Layer                    │
│  (Database, External APIs, File System, etc.)       │
└─────────────────────────────────────────────────────┘
```

### 1. Domain Layer (Доменный слой)

**Расположение:** `internal/domain/`

**Ответственность:**
- Бизнес-сущности (Entities)
- Интерфейсы репозиториев
- Бизнес-правила

**Правила:**
- ❌ НЕ зависит от других слоёв
- ❌ НЕ содержит технических деталей
- ✅ Содержит только бизнес-логику
- ✅ Определяет контракты (интерфейсы)

**Пример:**

```go
// internal/domain/entity/user.go
package entity

type User struct {
    ID        string
    Email     string
    Name      string
    CreatedAt time.Time
}

func (u *User) Validate() error {
    if u.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

// internal/domain/repository/user.go
package repository

type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    GetByID(ctx context.Context, id string) (*entity.User, error)
    // ...
}
```

### 2. Use Case Layer (Слой бизнес-логики)

**Расположение:** `internal/usecase/`

**Ответственность:**
- Оркестрация бизнес-логики
- Координация между репозиториями
- Применение бизнес-правил

**Правила:**
- ✅ Зависит только от Domain Layer
- ✅ Использует интерфейсы репозиториев
- ✅ Содержит сценарии использования (use cases)
- ❌ НЕ знает о технических деталях (HTTP, DB)

**Пример:**

```go
// internal/usecase/create_user.go
package usecase

type CreateUserUsecase struct {
    userRepo repository.UserRepository
    logger   *logger.Logger
}

func (u *CreateUserUsecase) Execute(ctx context.Context, input Input) (*entity.User, error) {
    // 1. Создать entity
    user := &entity.User{
        ID:    uuid.New().String(),
        Email: input.Email,
        Name:  input.Name,
    }
    
    // 2. Валидация бизнес-правил
    if err := user.Validate(); err != nil {
        return nil, err
    }
    
    // 3. Сохранение через repository
    if err := u.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }
    
    return user, nil
}
```

### 3. Delivery Layer (Слой доставки)

**Расположение:** `internal/delivery/http/`

**Ответственность:**
- Обработка HTTP запросов
- Парсинг и валидация входных данных
- Преобразование DTO ↔ Domain entities
- Обработка ошибок

**Правила:**
- ✅ Зависит от Use Case Layer
- ✅ Обрабатывает технические детали (HTTP)
- ✅ Преобразует данные между форматами
- ❌ НЕ содержит бизнес-логику

**Пример:**

```go
// internal/delivery/http/user.go
package http

type UserHandler struct {
    createUserUC *usecase.CreateUserUsecase
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
    // 1. Парсинг запроса
    var input CreateUserRequest
    if err := c.BodyParser(&input); err != nil {
        return errors.ErrBadRequest
    }
    
    // 2. Валидация входных данных
    if err := validate.Struct(input); err != nil {
        return errors.ErrValidation
    }
    
    // 3. Вызов use case
    user, err := h.createUserUC.Execute(ctx, usecase.CreateUserInput{
        Email: input.Email,
        Name:  input.Name,
    })
    if err != nil {
        return err
    }
    
    // 4. Преобразование и отправка ответа
    return c.Status(201).JSON(toUserResponse(user))
}
```

### 4. Infrastructure Layer (Инфраструктурный слой)

**Расположение:** `internal/infrastructure/`

**Ответственность:**
- Реализация интерфейсов репозиториев
- Работа с базами данных
- Вызовы внешних API
- Работа с файловой системой

**Правила:**
- ✅ Реализует интерфейсы из Domain Layer
- ✅ Содержит технические детали
- ❌ НЕ используется напрямую из Use Case

**Пример:**

```go
// internal/infrastructure/repository/user.go
package repository

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    query := `INSERT INTO users (id, email, name) VALUES ($1, $2, $3)`
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.Name)
    return err
}
```

## Dependency Injection с Uber FX

### Принципы

1. **Провайдеры** - функции, которые создают зависимости
2. **Инвокеры** - функции, которые используют зависимости
3. **Lifecycle** - управление жизненным циклом

### Пример регистрации

```go
func main() {
    app := fx.New(
        // Провайдеры (создание зависимостей)
        fx.Provide(
            // Инфраструктура
            provideConfig,
            logger.New,
            provideDatabase,
            
            // Repositories
            repository.NewUserRepository,
            repository.NewProductRepository,
            
            // Use Cases
            usecase.NewCreateUserUsecase,
            usecase.NewGetUserUsecase,
            
            // Handlers
            http.NewUserHandler,
        ),
        
        // Инвокеры (использование зависимостей)
        fx.Invoke(
            setupServer,
            registerRoutes,
        ),
    )
    
    app.Run()
}
```

### Автоматическое внедрение зависимостей

FX автоматически связывает зависимости:

```go
// Repository нуждается в DB
func NewUserRepository(db *sql.DB) repository.UserRepository

// Use Case нуждается в Repository
func NewCreateUserUsecase(
    repo repository.UserRepository,
    logger *logger.Logger,
) *CreateUserUsecase

// Handler нуждается в Use Case
func NewUserHandler(
    createUserUC *usecase.CreateUserUsecase,
) *UserHandler
```

FX создаст все зависимости в правильном порядке!

## Поток данных

### Создание пользователя (POST /users)

```
1. HTTP Request
   ↓
2. Handler.Create() - парсинг запроса
   ↓
3. CreateUserUsecase.Execute() - бизнес-логика
   ↓
4. UserRepository.Create() - сохранение в БД
   ↓
5. Database
   ↓
6. Response ← Handler ← Use Case ← Repository
```

### Детальный пример

```go
// 1. HTTP Request
POST /api/v1/users
{
  "email": "john@example.com",
  "name": "John Doe"
}

// 2. Handler
func (h *UserHandler) Create(c *fiber.Ctx) error {
    var req CreateUserRequest
    c.BodyParser(&req)
    
    // 3. Use Case
    user, err := h.createUserUC.Execute(ctx, usecase.CreateUserInput{
        Email: req.Email,
        Name:  req.Name,
    })
    
    return c.JSON(user)
}

// 4. Use Case
func (u *CreateUserUsecase) Execute(ctx context.Context, input Input) error {
    user := &entity.User{
        ID:    uuid.New().String(),
        Email: input.Email,
        Name:  input.Name,
    }
    
    user.Validate()
    
    // 5. Repository
    return u.userRepo.Create(ctx, user)
}

// 6. Repository
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    return r.db.Create(user).Error
}
```

## Best Practices

### 1. Один Use Case = Один сценарий

❌ Плохо:
```go
type UserUsecase struct {
    // Много методов
}
```

✅ Хорошо:
```go
type CreateUserUsecase struct { }
type GetUserUsecase struct { }
type UpdateUserUsecase struct { }
```

### 2. Интерфейсы в Domain, реализации в Infrastructure

❌ Плохо:
```go
// infrastructure/repository/user.go
type UserRepository interface { }
```

✅ Хорошо:
```go
// domain/repository/user.go
type UserRepository interface { }

// infrastructure/repository/user.go
type userRepository struct { }
```

### 3. Контекст везде

✅ Всегда передавайте context:
```go
func (u *Usecase) Execute(ctx context.Context) error
func (r *Repository) Create(ctx context.Context, user *User) error
```

### 4. Валидация на границах

```go
// Handler - валидация структуры запроса
func (h *Handler) Create(c *fiber.Ctx) error {
    validate.Struct(input)
}

// Entity - бизнес-правила
func (u *User) Validate() error {
    // Бизнес-логика
}
```

### 5. Возвращайте интерфейсы, принимайте конкретные типы

```go
// ✅ Хорошо
func NewUserRepository(db *sql.DB) repository.UserRepository {
    return &userRepository{db: db}
}

// ❌ Плохо
func NewUserRepository(db *sql.DB) *userRepository
```

## Тестирование

### Unit тесты Use Cases

```go
func TestCreateUserUsecase(t *testing.T) {
    // Mock repository
    mockRepo := &MockUserRepository{}
    
    // Create use case
    uc := usecase.NewCreateUserUsecase(mockRepo, logger)
    
    // Test
    user, err := uc.Execute(ctx, input)
    
    assert.NoError(t, err)
    assert.Equal(t, "john@example.com", user.Email)
}
```

### Integration тесты Handlers

```go
func TestUserHandler_Create(t *testing.T) {
    app := fiber.New()
    handler := http.NewUserHandler(...)
    handler.RegisterRoutes(app.Group("/api"))
    
    req := httptest.NewRequest("POST", "/api/users", body)
    resp, _ := app.Test(req)
    
    assert.Equal(t, 201, resp.StatusCode)
}
```

## Расширение архитектуры

### Добавление нового модуля

1. **Создать entity:**
```bash
microkit generate entity product
```

2. **Создать repository:**
```bash
microkit generate repository product
```

3. **Создать use cases:**
```bash
microkit generate usecase create-product
```

4. **Создать handler:**
```bash
microkit generate handler product
```

5. **Зарегистрировать в main.go:**
```go
fx.Provide(
    repository.NewProductRepository,
    usecase.NewCreateProductUsecase,
    http.NewProductHandler,
)
```

### Добавление нового типа доставки (например, gRPC)

```go
// internal/delivery/grpc/user.go
type UserGRPCHandler struct {
    createUserUC *usecase.CreateUserUsecase
}

func (h *UserGRPCHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
    user, err := h.createUserUC.Execute(ctx, usecase.CreateUserInput{
        Email: req.Email,
        Name:  req.Name,
    })
    
    return toProto(user), err
}
```

## Заключение

Чистая архитектура с Microkit:

- ✅ **Тестируемость** - легко писать unit тесты
- ✅ **Независимость от фреймворков** - можно заменить Fiber на что-то другое
- ✅ **Независимость от БД** - можно переключиться с Postgres на MongoDB
- ✅ **Независимость от UI** - HTTP, gRPC, CLI используют одни use cases
- ✅ **Бизнес-логика в центре** - domain не зависит ни от чего

**Помните:** Зависимости направлены внутрь к Domain Layer!