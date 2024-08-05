package fiber_server

import "github.com/gofiber/fiber/v2"

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

}
