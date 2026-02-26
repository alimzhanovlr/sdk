package middleware

import (
	"time"

	"github.com/alimzhanovlr/sdk/logger"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// LoggerMiddleware adds logging to requests
func LoggerMiddleware(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get trace ID if available
		traceID, _ := c.Locals("trace_id").(string)

		// Continue with request
		err := c.Next()

		// Log request
		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("Request failed", fields...)
		} else {
			log.Info("Request completed", fields...)
		}

		return err
	}
}
