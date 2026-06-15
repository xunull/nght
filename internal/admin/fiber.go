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

// Lookup returns the RouteConfig registered for path on the package-level
// default table, or (RouteConfig{}, false) if not present. Used by
// fiber_server's wildcard middleware to dispatch dynamic routes without
// exposing the table singleton to other packages.
func Lookup(path string) (RouteConfig, bool) {
	return defaultTable.Get(path)
}

// MarkReserved marks path as reserved on the package-level default table.
// The fiber_server's route setup calls this for every hardcoded
// endpoint so dynamic Register calls cannot shadow built-in behavior
// (plan-eng-review C1 single source of truth).
func MarkReserved(path string) {
	defaultTable.MarkReserved(path)
}

// Register adds a route to the package-level default table. Returns
// nil on success or one of the package-level validation errors.
// Used by integration tests and by future CLI / control-plane clients
// that want to bypass the /admin/routes HTTP API and seed the table
// at startup.
func Register(cfg RouteConfig) error {
	return defaultTable.Register(cfg)
}

// Unregister removes a route from the package-level default table.
// Returns true if it existed. Idempotent.
func Unregister(path string) bool {
	return defaultTable.Unregister(path)
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
