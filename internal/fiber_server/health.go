package fiber_server

import (
	"github.com/gofiber/fiber/v2"
	"sync"
)

var (
	healthStatus = true
	healthMu     sync.Mutex
)

func HealthResp(c *fiber.Ctx) error {

	return c.JSON(fiber.Map{
		"status": "UP",
	})
}

func HealthRandomResp(c *fiber.Ctx) error {

	c.Params("time")

	return c.JSON(fiber.Map{
		"status": "UP",
	})
}

func SetHealthTrue(c *fiber.Ctx) error {
	healthMu.Lock()
	defer healthMu.Unlock()
	healthStatus = true

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": healthStatus,
	})
}

func SetHealthFalse(c *fiber.Ctx) error {
	healthMu.Lock()
	defer healthMu.Unlock()
	healthStatus = false

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": healthStatus,
	})
}
