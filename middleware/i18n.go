package middleware

import (
	"strings"

	"github.com/alimzhanovlr/sdk/i18n"
	"github.com/gofiber/fiber/v2"
)

// I18nMiddleware adds i18n support to requests
func I18nMiddleware(i18nInstance *i18n.I18n) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get language from header or query
		lang := c.Get("Accept-Language")
		if queryLang := c.Query("lang"); queryLang != "" {
			lang = queryLang
		}

		// Extract first language from Accept-Language header
		if lang != "" {
			parts := strings.Split(lang, ",")
			if len(parts) > 0 {
				lang = strings.TrimSpace(strings.Split(parts[0], ";")[0])
			}
		}

		// Validate language
		if !i18nInstance.IsSupported(lang) {
			lang = ""
		}

		// Store language in context
		c.Locals("lang", lang)

		return c.Next()
	}
}

// GetLanguage extracts language from context
func GetLanguage(c *fiber.Ctx) string {
	if lang, ok := c.Locals("lang").(string); ok && lang != "" {
		return lang
	}
	return "en"
}
