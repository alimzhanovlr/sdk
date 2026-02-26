package server

import (
	"context"
	"fmt"
	"time"

	"github.com/alimzhanovlr/sdk/config"
	"github.com/alimzhanovlr/sdk/logger"
	"github.com/alimzhanovlr/sdk/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"
)

// Server wraps Fiber app
type Server struct {
	app    *fiber.App
	config config.ServerConfig
	logger *logger.Logger
	tracer *tracing.Tracer
}

// Params for server constructor
type Params struct {
	fx.In

	Config *config.Config
	Logger *logger.Logger
	Tracer *tracing.Tracer
}

// New creates a new server
func New(p Params) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  time.Duration(p.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(p.Config.Server.WriteTimeout) * time.Second,
		ErrorHandler: errorHandler(p.Logger),
	})

	// Add recover middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	return &Server{
		app:    app,
		config: p.Config.Server,
		logger: p.Logger,
		tracer: p.Tracer,
	}
}

// App returns Fiber app
func (s *Server) App() *fiber.App {
	return s.app
}

// Start starts the server
func (s *Server) Start(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
			s.logger.Info("Starting server",
				logger.String("address", addr),
			)

			go func() {
				if err := s.app.Listen(addr); err != nil {
					s.logger.Error("Failed to start server", logger.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("Shutting down server")
			return s.app.Shutdown()
		},
	})
}

// RegisterRoutes registers route handler
func (s *Server) RegisterRoutes(register func(*fiber.App)) {
	register(s.app)
}

// errorHandler handles Fiber errors
func errorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		log.Error("Request error",
			logger.String("method", c.Method()),
			logger.String("path", c.Path()),
			logger.Int("status", code),
			logger.Error(err),
		)

		return c.Status(code).JSON(fiber.Map{
			"error": fiber.Map{
				"message": message,
				"code":    code,
			},
		})
	}
}
