package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate code components",
		Aliases: []string{"gen", "g"},
	}

	cmd.AddCommand(
		newGenerateEntityCmd(),
		newGenerateUsecaseCmd(),
		newGenerateHandlerCmd(),
		newGenerateRepositoryCmd(),
	)

	return cmd
}

func newGenerateEntityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "entity [name]",
		Short: "Generate a domain entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateEntity(args[0])
		},
	}
}

func newGenerateUsecaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usecase [name]",
		Short: "Generate a use case",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateUsecase(args[0])
		},
	}
}

func newGenerateHandlerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "handler [name]",
		Short: "Generate an HTTP handler",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateHandler(args[0])
		},
	}
}

func newGenerateRepositoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repository [name]",
		Short: "Generate a repository interface and implementation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateRepository(args[0])
		},
	}
}

func generateEntity(name string) error {
	entityName := toPascalCase(name)
	fileName := toSnakeCase(name) + ".go"

	data := struct {
		Name string
	}{Name: entityName}

	dir := "internal/domain/entity"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, fileName)
	if err := generateFile(path, entityTemplate, data); err != nil {
		return err
	}

	fmt.Printf("✅ Generated entity: %s\n", path)
	return nil
}

func generateUsecase(name string) error {
	usecaseName := toPascalCase(name)
	fileName := toSnakeCase(name) + ".go"

	data := struct {
		Name    string
		VarName string
	}{
		Name:    usecaseName,
		VarName: toLowerCamelCase(name),
	}

	dir := "internal/usecase"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, fileName)
	if err := generateFile(path, usecaseTemplate, data); err != nil {
		return err
	}

	fmt.Printf("✅ Generated usecase: %s\n", path)
	return nil
}

func generateHandler(name string) error {
	handlerName := toPascalCase(name)
	fileName := toSnakeCase(name) + ".go"

	data := struct {
		Name    string
		VarName string
	}{
		Name:    handlerName,
		VarName: toLowerCamelCase(name),
	}

	dir := "internal/delivery/http"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, fileName)
	if err := generateFile(path, handlerTemplate, data); err != nil {
		return err
	}

	fmt.Printf("✅ Generated handler: %s\n", path)
	return nil
}

func generateRepository(name string) error {
	repoName := toPascalCase(name)
	fileName := toSnakeCase(name) + ".go"

	data := struct {
		Name    string
		VarName string
	}{
		Name:    repoName,
		VarName: toLowerCamelCase(name),
	}

	// Generate interface
	interfaceDir := "internal/domain/repository"
	if err := os.MkdirAll(interfaceDir, 0755); err != nil {
		return err
	}

	interfacePath := filepath.Join(interfaceDir, fileName)
	if err := generateFile(interfacePath, repositoryInterfaceTemplate, data); err != nil {
		return err
	}

	// Generate implementation
	implDir := "internal/infrastructure/repository"
	if err := os.MkdirAll(implDir, 0755); err != nil {
		return err
	}

	implPath := filepath.Join(implDir, fileName)
	if err := generateFile(implPath, repositoryImplTemplate, data); err != nil {
		return err
	}

	fmt.Printf("✅ Generated repository interface: %s\n", interfacePath)
	fmt.Printf("✅ Generated repository implementation: %s\n", implPath)
	return nil
}

// Utility functions
func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	return strings.Join(words, "")
}

func toLowerCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Templates
const entityTemplate = `package entity

import "time"

// {{.Name}} represents a {{.Name}} entity
type {{.Name}} struct {
	ID        string    ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
	
	// TODO: Add your fields here
}

// Validate validates the {{.Name}} entity
func (e *{{.Name}}) Validate() error {
	// TODO: Implement validation
	return nil
}
`

const usecaseTemplate = `package usecase

import (
	"context"
	
	"github.com/yourorg/microkit/pkg/logger"
	"github.com/yourorg/microkit/pkg/tracing"
)

// {{.Name}}Usecase handles {{.Name}} business logic
type {{.Name}}Usecase struct {
	logger *logger.Logger
	tracer *tracing.Tracer
	// TODO: Add repository dependencies
}

// New{{.Name}}Usecase creates a new {{.Name}}Usecase
func New{{.Name}}Usecase(
	logger *logger.Logger,
	tracer *tracing.Tracer,
) *{{.Name}}Usecase {
	return &{{.Name}}Usecase{
		logger: logger,
		tracer: tracer,
	}
}

// Execute executes the use case
func (u *{{.Name}}Usecase) Execute(ctx context.Context) error {
	ctx, span := u.tracer.Start(ctx, "{{.Name}}Usecase.Execute")
	defer span.End()
	
	u.logger.Info("Executing {{.Name}} use case")
	
	// TODO: Implement business logic
	
	return nil
}
`

const handlerTemplate = `package http

import (
	"github.com/gofiber/fiber/v2"
	
	"github.com/yourorg/microkit/pkg/logger"
	"github.com/yourorg/microkit/pkg/errors"
	"github.com/yourorg/microkit/pkg/middleware"
)

// {{.Name}}Handler handles {{.Name}} HTTP requests
type {{.Name}}Handler struct {
	logger *logger.Logger
	// TODO: Add usecase dependencies
}

// New{{.Name}}Handler creates a new {{.Name}}Handler
func New{{.Name}}Handler(logger *logger.Logger) *{{.Name}}Handler {
	return &{{.Name}}Handler{
		logger: logger,
	}
}

// RegisterRoutes registers {{.Name}} routes
func (h *{{.Name}}Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/{{.VarName}}")
	
	group.Get("/", h.List)
	group.Get("/:id", h.Get)
	group.Post("/", h.Create)
	group.Put("/:id", h.Update)
	group.Delete("/:id", h.Delete)
}

// List handles GET /{{.VarName}}
func (h *{{.Name}}Handler) List(c *fiber.Ctx) error {
	ctx := c.UserContext()
	lang := middleware.GetLanguage(c)
	
	h.logger.Info("Listing {{.VarName}}",
		logger.String("lang", lang),
	)
	
	// TODO: Implement list logic
	
	return c.JSON(fiber.Map{
		"data": []interface{}{},
	})
}

// Get handles GET /{{.VarName}}/:id
func (h *{{.Name}}Handler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// TODO: Implement get logic
	
	return c.JSON(fiber.Map{
		"id": id,
	})
}

// Create handles POST /{{.VarName}}
func (h *{{.Name}}Handler) Create(c *fiber.Ctx) error {
	// TODO: Parse request body
	// TODO: Validate
	// TODO: Call use case
	
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Created successfully",
	})
}

// Update handles PUT /{{.VarName}}/:id
func (h *{{.Name}}Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// TODO: Parse request body
	// TODO: Validate
	// TODO: Call use case
	
	return c.JSON(fiber.Map{
		"id": id,
		"message": "Updated successfully",
	})
}

// Delete handles DELETE /{{.VarName}}/:id
func (h *{{.Name}}Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// TODO: Call use case
	
	return c.JSON(fiber.Map{
		"id": id,
		"message": "Deleted successfully",
	})
}
`

const repositoryInterfaceTemplate = `package repository

import (
	"context"
	
	"your-module/internal/domain/entity"
)

// {{.Name}}Repository defines {{.Name}} data access interface
type {{.Name}}Repository interface {
	// Create creates a new {{.Name}}
	Create(ctx context.Context, {{.VarName}} *entity.{{.Name}}) error
	
	// GetByID retrieves a {{.Name}} by ID
	GetByID(ctx context.Context, id string) (*entity.{{.Name}}, error)
	
	// Update updates an existing {{.Name}}
	Update(ctx context.Context, {{.VarName}} *entity.{{.Name}}) error
	
	// Delete deletes a {{.Name}} by ID
	Delete(ctx context.Context, id string) error
	
	// List retrieves all {{.Name}}s with pagination
	List(ctx context.Context, limit, offset int) ([]*entity.{{.Name}}, error)
}
`

const repositoryImplTemplate = `package repository

import (
	"context"
	"fmt"
	
	"your-module/internal/domain/entity"
	"your-module/internal/domain/repository"
	
	"github.com/yourorg/microkit/pkg/logger"
	"github.com/yourorg/microkit/pkg/tracing"
	"github.com/yourorg/microkit/pkg/errors"
)

// {{.VarName}}Repository implements {{.Name}}Repository interface
type {{.VarName}}Repository struct {
	logger *logger.Logger
	tracer *tracing.Tracer
	// TODO: Add database connection
}

// New{{.Name}}Repository creates a new {{.Name}}Repository
func New{{.Name}}Repository(
	logger *logger.Logger,
	tracer *tracing.Tracer,
) repository.{{.Name}}Repository {
	return &{{.VarName}}Repository{
		logger: logger,
		tracer: tracer,
	}
}

// Create creates a new {{.Name}}
func (r *{{.VarName}}Repository) Create(ctx context.Context, {{.VarName}} *entity.{{.Name}}) error {
	ctx, span := r.tracer.Start(ctx, "{{.Name}}Repository.Create")
	defer span.End()
	
	r.logger.Info("Creating {{.VarName}}")
	
	// TODO: Implement database insert
	
	return nil
}

// GetByID retrieves a {{.Name}} by ID
func (r *{{.VarName}}Repository) GetByID(ctx context.Context, id string) (*entity.{{.Name}}, error) {
	ctx, span := r.tracer.Start(ctx, "{{.Name}}Repository.GetByID")
	defer span.End()
	
	r.logger.Info("Getting {{.VarName}} by ID", logger.String("id", id))
	
	// TODO: Implement database query
	
	return nil, errors.ErrNotFound
}

// Update updates an existing {{.Name}}
func (r *{{.VarName}}Repository) Update(ctx context.Context, {{.VarName}} *entity.{{.Name}}) error {
	ctx, span := r.tracer.Start(ctx, "{{.Name}}Repository.Update")
	defer span.End()
	
	r.logger.Info("Updating {{.VarName}}")
	
	// TODO: Implement database update
	
	return nil
}

// Delete deletes a {{.Name}} by ID
func (r *{{.VarName}}Repository) Delete(ctx context.Context, id string) error {
	ctx, span := r.tracer.Start(ctx, "{{.Name}}Repository.Delete")
	defer span.End()
	
	r.logger.Info("Deleting {{.VarName}}", logger.String("id", id))
	
	// TODO: Implement database delete
	
	return nil
}

// List retrieves all {{.Name}}s with pagination
func (r *{{.VarName}}Repository) List(ctx context.Context, limit, offset int) ([]*entity.{{.Name}}, error) {
	ctx, span := r.tracer.Start(ctx, "{{.Name}}Repository.List")
	defer span.End()
	
	r.logger.Info("Listing {{.VarName}}s", 
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)
	
	// TODO: Implement database query with pagination
	
	return []*entity.{{.Name}}{}, nil
}
`
