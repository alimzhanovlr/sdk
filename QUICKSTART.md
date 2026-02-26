package server

import (
"github.com/gofiber/fiber/v2"
"github.com/yourorg/microkit/pkg/errors"
)

// Response represents a standard API response
type Response struct {
Success bool        `json:"success"`
Data    interface{} `json:"data,omitempty"`
Error   *ErrorInfo  `json:"error,omitempty"`
Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
Code    string                 `json:"code"`
Message string                 `json:"message"`
Details map[string]interface{} `json:"details,omitempty"`
}

// Meta represents response metadata
type Meta struct {
Page       int `json:"page,omitempty"`
PerPage    int `json:"per_page,omitempty"`
Total      int `json:"total,omitempty"`
TotalPages int `json:"total_pages,omitempty"`
}

// SendSuccess sends a success response
func SendSuccess(c *fiber.Ctx, data interface{}) error {
return c.JSON(Response{
Success: true,
Data:    data,
})
}

// SendSuccessWithMeta sends a success response with metadata
func SendSuccessWithMeta(c *fiber.Ctx, data interface{}, meta *Meta) error {
return c.JSON(Response{
Success: true,
Data:    data,
Meta:    meta,
})
}

// SendCreated sends a 201 Created response
func SendCreated(c *fiber.Ctx, data interface{}) error {
return c.Status(fiber.StatusCreated).JSON(Response{
Success: true,
Data:    data,
})
}

// SendNoContent sends a 204 No Content response
func SendNoContent(c *fiber.Ctx) error {
return c.SendStatus(fiber.StatusNoContent)
}

// SendError sends an error response
func SendError(c *fiber.Ctx, err error) error {
appErr := errors.GetAppError(err)

	return c.Status(appErr.StatusCode).JSON(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}

// SendCustomError sends a custom error response
func SendCustomError(c *fiber.Ctx, statusCode int, code, message string) error {
return c.Status(statusCode).JSON(Response{
Success: false,
Error: &ErrorInfo{
Code:    code,
Message: message,
},
})
}

// CalculateMeta calculates pagination metadata
func CalculateMeta(page, perPage, total int) *Meta {
totalPages := total / perPage
if total%perPage > 0 {
totalPages++
}

	return &Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}