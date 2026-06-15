package admin

import (
	"github.com/gofiber/fiber/v2"
)

const routesPrefix = "/admin/routes"

// defaultTable is the package-level RouteTable used by RegisterFiberRoutes.
// Tests inject a fresh table via RegisterFiberRoutesWithTable for isolation.
var defaultTable = NewRouteTable()

// RegisterFiberRoutes mounts the /admin/routes API on app, guarded by
// RequireAdminToken(secret). Routes:
//
//	POST   /admin/routes        register a new dynamic route
//	GET    /admin/routes        list all registered routes
//	DELETE /admin/routes/<path> unregister a route (path is the
//	                            full URL suffix after /admin/routes/)
//
// Returns 201/204/200/400/401/404 as appropriate. Body is always JSON.
func RegisterFiberRoutes(app *fiber.App, secret string) {
	RegisterFiberRoutesWithTable(app, secret, defaultTable)
}

// RegisterFiberRoutesWithTable is the same as RegisterFiberRoutes but
// accepts an explicit *RouteTable. Use this in tests for isolation.
func RegisterFiberRoutesWithTable(app *fiber.App, secret string, tbl *RouteTable) {
	mw := RequireAdminToken(secret)
	app.Post(routesPrefix, mw, func(c *fiber.Ctx) error { return handleRegister(c, tbl) })
	app.Get(routesPrefix, mw, func(c *fiber.Ctx) error { return handleList(c, tbl) })
	app.All(routesPrefix+"/*", mw, func(c *fiber.Ctx) error { return handleUnregister(c, tbl) })
}

func handleRegister(c *fiber.Ctx, tbl *RouteTable) error {
	var cfg RouteConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON: " + err.Error(),
		})
	}
	if err := tbl.Register(cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(cfg)
}

func handleList(c *fiber.Ctx, tbl *RouteTable) error {
	return c.Status(fiber.StatusOK).JSON(tbl.List())
}

func handleUnregister(c *fiber.Ctx, tbl *RouteTable) error {
	raw := c.Params("*")
	if raw == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "path is empty",
		})
	}
	path := "/" + raw
	_ = tbl.Unregister(path)
	return c.SendStatus(fiber.StatusNoContent)
}
