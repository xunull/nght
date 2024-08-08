package fiber_server

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/xunull/nght/internal/global"
)

func SetupRoutes(app *fiber.App) {

	app.Use(func(c *fiber.Ctx) error {
		SetCommonHeader(c)
		return c.Next()
	})

	app.All("/echo/:text", EchoTextResp)

	app.All("/status/:status", StatusResp)

	app.All("/response_time/:time", ResponseTimeResp)

	app.All("/random/:statusRandom", RandomStatusResp)

	app.All("/random_crash/:percentage/:statusRandom", RandomCrashResp)

	healthGroup := app.Group("/health")
	{
		healthGroup.All("", HealthResp)
		healthGroup.All("/random/:percentage", HealthRandomResp)
		healthGroup.All("/true", SetHealthTrue)
		healthGroup.All("/false", SetHealthFalse)
	}

	// 顺序很重要，不能放在上面的路由之前
	app.All("*", func(c *fiber.Ctx) error {
		fmt.Println(c.OriginalURL())

		if responseJsonFlag {
			return c.JSON(fiber.Map{
				"url":      c.OriginalURL(),
				"hostname": global.Hostname,
			})
		} else {
			return c.SendString(fmt.Sprintf("url: %s\nhostname: %s\n", c.OriginalURL(), global.Hostname))
		}
	})

}
