package admin

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

// RequireAdminToken returns a fiber middleware that enforces NGHT_ADMIN_TOKEN.
//
//   - secret == "":  allow all (the opt-in case — server has no admin
//     protection because the operator chose not to set it)
//   - secret != "":  require X-Admin-Token header with exact-match (no
//     trim, no normalization). Mismatch returns 401.
//
// Compare is constant-time for same-length tokens. Length-mismatch
// returns 401 immediately, which leaks the configured length — an
// accepted tradeoff (admin tokens are typically fixed-length random
// strings, and the practical attack is brute-force, not length oracle).
//
// Boot-time whitespace check on NGHT_ADMIN_TOKEN itself is the caller's
// responsibility (see cmd/server.go), not this middleware's.
func RequireAdminToken(secret string) fiber.Handler {
	if secret == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}
	return func(c *fiber.Ctx) error {
		header := c.Request().Header.Peek("X-Admin-Token")
		if len(header) != len(secret) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		if subtle.ConstantTimeCompare(header, []byte(secret)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		return c.Next()
	}
}
