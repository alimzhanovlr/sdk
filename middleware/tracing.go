package middleware

import (
	"github.com/alimzhanovlr/sdk/tracing"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
)

// TracingMiddleware adds tracing to requests
func TracingMiddleware(tracer *tracing.Tracer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// Start span
		spanName := c.Method() + " " + c.Route().Path
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		// Add attributes
		span.SetAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.url", c.OriginalURL()),
			attribute.String("http.route", c.Route().Path),
		)

		// Store trace ID in context
		traceID := tracing.GetTraceID(ctx)
		c.Locals("trace_id", traceID)
		c.Set("X-Trace-ID", traceID)

		// Continue with request
		c.SetUserContext(ctx)
		err := c.Next()

		// Record status
		span.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))

		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}
