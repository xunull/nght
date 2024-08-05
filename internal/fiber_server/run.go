package fiber_server

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Server(port int) {
	app := fiber.New(
		fiber.Config{
			CaseSensitive:     false,
			DisableKeepalive:  false,
			EnablePrintRoutes: true,
			Network:           fiber.NetworkTCP4,
		})

	app.Use(recover.New(
		recover.Config{
			EnableStackTrace: true,
		}))
	app.Use(logger.New())
	app.Use(requestid.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	SetupRoutes(app)

	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			log.Panic(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c

	_ = app.Shutdown()

	// Your cleanup tasks go here
}
