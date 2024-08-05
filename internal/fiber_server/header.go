package fiber_server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/xunull/nght/internal/global"
)

func SetCommonHeader(c *fiber.Ctx) {
	c.Response().Header.Set("NGHT-Hostname", global.Hostname)
}
