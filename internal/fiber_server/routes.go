package fiber_server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/xunull/nght/internal/admin"
	"github.com/xunull/nght/internal/global"
)

func SetupRoutes(app *fiber.App, adminToken string) {

	app.Use(func(c *fiber.Ctx) error {
		SetCommonHeader(c)
		return c.Next()
	})

	// Mark all hardcoded paths as reserved before registering them.
	// This is the single source of truth (plan-eng-review C1): a
	// dynamic-route Register of any of these paths is rejected.
	// Prefix reservations use a trailing "/" so they cover all
	// sub-paths under the prefix. Exact reservations cover named
	// leaf paths (e.g., "/echo_header", "/livez", "/healthz").
	admin.MarkReserved("/echo")  // exact — covers "/echo" and namespace
	admin.MarkReserved("/echo/") // prefix — covers "/echo/:text", "/echo_url", etc.
	admin.MarkReserved("/echo_header")
	admin.MarkReserved("/echo_url")
	admin.MarkReserved("/status")
	admin.MarkReserved("/status/")
	admin.MarkReserved("/log_req_data")
	admin.MarkReserved("/response_time")
	admin.MarkReserved("/response_time/")
	admin.MarkReserved("/random")
	admin.MarkReserved("/random/")
	admin.MarkReserved("/random_crash")
	admin.MarkReserved("/random_crash/")
	admin.MarkReserved("/healthz")
	admin.MarkReserved("/livez")
	admin.MarkReserved("/health/")

	app.All("/echo/:text", EchoTextResp)
	app.All("/echo_header", EchoReqHeader)
	app.All("/echo_url", EchoUrlResp)
	app.All("/status/:status", StatusResp)
	app.All("/log_req_data", LogReqData)

	app.All("/response_time/:time", ResponseTimeResp)
	app.All("/random/:statusRandom", RandomStatusResp)
	app.All("/random_crash/:percentage/:statusRandom", RandomCrashResp)

	app.All("/healthz", HealthResp)
	app.All("/livez", LivezResp)

	healthGroup := app.Group("/health")
	{
		healthGroup.All("", HealthResp)
		healthGroup.All("/random/:percentage", HealthRandomResp)
		healthGroup.All("/true", SetHealthTrue)
		healthGroup.All("/false", SetHealthFalse)
	}

	// Register admin routes BEFORE the wildcard catch-all below.
	// Empirically fiber's radix tree in real Listen mode routes `*`
	// before later-registered specific paths if the wildcard was
	// registered first; moving admin.RegisterFiberRoutes here fixes
	// that ordering so the more-specific admin paths win.
	admin.RegisterFiberRoutes(app, adminToken)

	// Wildcard fallback. Dynamic routes (admin table) take priority;
	// miss falls through to the original behavior of returning a 200
	// with the url + hostname for unrecognized paths.
	app.All("*", func(c *fiber.Ctx) error {
		if cfg, ok := admin.Lookup(c.Path()); ok {
			if cfg.LatencyMs > 0 {
				time.Sleep(time.Duration(cfg.LatencyMs) * time.Millisecond)
			}
			return c.SendStatus(cfg.StatusCode)
		}

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
