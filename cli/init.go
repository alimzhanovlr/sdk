package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		projectName string
		modulePath  string
	)

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new microservice project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName = args[0]

			if modulePath == "" {
				modulePath = "github.com/yourorg/" + projectName
			}

			return initProject(projectName, modulePath)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "", "Go module path")

	return cmd
}

func initProject(projectName, modulePath string) error {
	fmt.Printf("Initializing project: %s\n", projectName)
	fmt.Printf("Module path: %s\n", modulePath)

	// Create project structure
	dirs := []string{
		projectName,
		filepath.Join(projectName, "cmd", "api"),
		filepath.Join(projectName, "internal", "domain", "entity"),
		filepath.Join(projectName, "internal", "domain", "repository"),
		filepath.Join(projectName, "internal", "usecase"),
		filepath.Join(projectName, "internal", "delivery", "http"),
		filepath.Join(projectName, "internal", "infrastructure", "repository"),
		filepath.Join(projectName, "config"),
		filepath.Join(projectName, "locales"),
		filepath.Join(projectName, "migrations"),
		filepath.Join(projectName, "scripts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		filepath.Join(projectName, "go.mod"):                goModTemplate,
		filepath.Join(projectName, "cmd", "api", "main.go"): mainTemplate,
		filepath.Join(projectName, "config", "config.yaml"): configTemplate,
		filepath.Join(projectName, "locales", "en.yaml"):    enLocaleTemplate,
		filepath.Join(projectName, "locales", "ru.yaml"):    ruLocaleTemplate,
		filepath.Join(projectName, "README.md"):             readmeTemplate,
		filepath.Join(projectName, "Makefile"):              makefileTemplate,
		filepath.Join(projectName, ".gitignore"):            gitignoreTemplate,
		filepath.Join(projectName, "Dockerfile"):            dockerfileTemplate,
	}

	data := struct {
		ProjectName string
		ModulePath  string
	}{
		ProjectName: projectName,
		ModulePath:  modulePath,
	}

	for path, tmpl := range files {
		if err := generateFile(path, tmpl, data); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", path)
	}

	fmt.Printf("\n✅ Project %s initialized successfully!\n", projectName)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  go mod tidy")
	fmt.Println("  make run")

	return nil
}

func generateFile(path, tmplStr string, data interface{}) error {
	tmpl, err := template.New(filepath.Base(path)).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

const goModTemplate = `module {{.ModulePath}}

go 1.22

require (
	github.com/yourorg/microkit v1.0.0
)
`

const mainTemplate = `package main

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

func setupServer(lc fx.Lifecycle, srv *server.Server, log *logger.Logger, tracer *tracing.Tracer, i18n *i18n.I18n) {
	// Add middleware
	srv.App().Use(middleware.TracingMiddleware(tracer))
	srv.App().Use(middleware.LoggerMiddleware(log))
	srv.App().Use(middleware.I18nMiddleware(i18n))
	
	// Start server
	srv.Start(lc)
	
	// Shutdown tracer
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return tracer.Shutdown(ctx)
		},
	})
}

func registerRoutes(srv *server.Server, log *logger.Logger) {
	app := srv.App()
	
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})
	
	// API routes
	api := app.Group("/api/v1")
	
	// TODO: Register your routes here
	api.Get("/example", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello from {{.ProjectName}}!",
		})
	})
}
`

const configTemplate = `server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30
  write_timeout: 30

logger:
  level: info
  format: json
  output_path: stdout

tracing:
  enabled: false
  service_name: {{.ProjectName}}
  endpoint: http://localhost:14268/api/traces
  sample_rate: 1.0

i18n:
  default_language: en
  supported_languages:
    - en
    - ru
  path: ./locales
`

const enLocaleTemplate = `welcome:
  other: "Welcome"
  
hello:
  other: "Hello, {{.Name}}!"
  
error:
  not_found: "Resource not found"
  internal: "Internal server error"
`

const ruLocaleTemplate = `welcome:
  other: "Добро пожаловать"
  
hello:
  other: "Привет, {{.Name}}!"
  
error:
  not_found: "Ресурс не найден"
  internal: "Внутренняя ошибка сервера"
`

const readmeTemplate = `# {{.ProjectName}}

Microservice built with Microkit framework and clean architecture.

## Getting Started

### Prerequisites

- Go 1.22+
- Make

### Installation

` + "```bash" + `
go mod tidy
` + "```" + `

### Running

` + "```bash" + `
make run
` + "```" + `

### Development

` + "```bash" + `
# Run with hot reload
make dev

# Run tests
make test

# Build
make build
` + "```" + `

## Project Structure

` + "```" + `
.
├── cmd/
│   └── api/           # Application entry point
├── internal/
│   ├── domain/        # Business logic and entities
│   │   ├── entity/    # Domain entities
│   │   └── repository/# Repository interfaces
│   ├── usecase/       # Use cases (business logic)
│   ├── delivery/      # Delivery layer (HTTP, gRPC, etc.)
│   │   └── http/      # HTTP handlers
│   └── infrastructure/# Infrastructure layer
│       └── repository/# Repository implementations
├── config/            # Configuration files
├── locales/           # i18n translations
└── migrations/        # Database migrations
` + "```" + `

## API Documentation

### Endpoints

- ` + "`GET /health`" + ` - Health check
- ` + "`GET /api/v1/example`" + ` - Example endpoint

## Configuration

Configuration is managed through ` + "`config/config.yaml`" + ` and environment variables.

Environment variables override config file values:
- ` + "`APP_SERVER_PORT`" + ` - Server port
- ` + "`APP_LOGGER_LEVEL`" + ` - Log level
- ` + "`APP_TRACING_ENABLED`" + ` - Enable tracing

## License

MIT
`

const makefileTemplate = `.PHONY: run build test clean dev

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/

dev:
	air -c .air.toml
`

const gitignoreTemplate = `# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Config overrides
config/local.yaml
`

const dockerfileTemplate = `FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/api .
COPY --from=builder /app/config ./config
COPY --from=builder /app/locales ./locales

EXPOSE 8080

CMD ["./api"]
`
