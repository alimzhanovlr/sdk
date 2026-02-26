package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Max        int           // Maximum number of requests
	Expiration time.Duration // Time window
	Message    string        // Error message
}

// DefaultRateLimitConfig returns default rate limit config
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:        100,
		Expiration: 1 * time.Minute,
		Message:    "Too many requests, please try again later",
	}
}

// RateLimitMiddleware returns rate limiting middleware
func RateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.Max,
		Expiration: config.Expiration,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "rate_limit_exceeded",
					"message": config.Message,
				},
			})
		},
	})
}
