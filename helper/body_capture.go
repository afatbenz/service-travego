package helper

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
)

func BodyCaptureMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			body := c.Body()
			if len(body) > 0 {
				bodyCopy := make([]byte, len(body))
				copy(bodyCopy, body)
				c.Locals("request_body", bodyCopy)
				c.Request().SetBodyStream(bytes.NewReader(body), len(body))
			}
		}
		return c.Next()
	}
}
