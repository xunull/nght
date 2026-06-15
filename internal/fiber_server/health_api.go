package fiber_server

import (
	"github.com/gofiber/fiber/v2"
	"math/rand"
	"net/http"
	"sync"
)

var (
	healthStatus = true
	healthMu     sync.Mutex
)

func HealthResp(c *fiber.Ctx) error {
	if healthStatus {
		return c.JSON(fiber.Map{
			"status": "UP",
		})
	} else {
		return c.SendStatus(http.StatusBadGateway)
	}
}

func HealthRandomResp(c *fiber.Ctx) error {
	percentage, err := c.ParamsInt("percentage")
	if err != nil {
		return err
	}
	if rand.Intn(100) < percentage {
		return c.JSON(fiber.Map{
			"status": "UP",
		})
	}
	return c.SendStatus(http.StatusBadGateway)
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
