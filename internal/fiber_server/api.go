package fiber_server

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/xunull/nght/internal/global"
	"github.com/xunull/nght/internal/utils"
	"math/rand"
	"time"
)

func EchoTextResp(c *fiber.Ctx) error {

	if responseJsonFlag {
		return c.JSON(fiber.Map{
			"text":     c.Params("text"),
			"hostname": global.Hostname,
		})
	} else {
		return c.SendString(fmt.Sprintf("text: %s\nhostname: %s\n", c.Params("text"), global.Hostname))
	}

}

func StatusResp(c *fiber.Ctx) error {
	status, err := c.ParamsInt("status")
	if err != nil {
		return err
	}
	if responseJsonFlag {
		return c.Status(status).JSON(fiber.Map{
			"status": c.Params("status"),
		})
	} else {
		return c.SendString(fmt.Sprintf("status: %s\n", status))
	}

}

func ResponseTimeResp(c *fiber.Ctx) error {
	responseTime, err := c.ParamsInt("time")
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(responseTime))
	if responseJsonFlag {
		return c.JSON(fiber.Map{
			"time": c.Params("time"),
		})
	} else {
		return c.SendString(fmt.Sprintf("time: %s\n", responseTime))
	}

}

func RandomStatusResp(c *fiber.Ctx) error {
	statusRandom := c.Params("statusRandom")
	if status, err := utils.SplitStatus(statusRandom); err != nil {
		return err
	} else {
		t := time.Now().UnixNano()
		rs := status[0]
		if len(status) > 1 {
			i := t % int64(len(status))

			rs = status[i]

		}
		if responseJsonFlag {
			return c.Status(rs).JSON(fiber.Map{
				"status": rs,
			})
		} else {
			return c.SendString(fmt.Sprintf("status: %d\n", rs))
		}
	}

}

func RandomCrashResp(c *fiber.Ctx) error {
	percentage, err := c.ParamsInt("percentage")
	if err != nil {
		return err
	}
	statusRandom := c.Params("statusRandom")

	if rand.Intn(100) > percentage {
		return c.SendStatus(fiber.StatusOK)
	}
	if status, err := utils.SplitStatus(statusRandom); err != nil {
		return err
	} else {
		t := time.Now().UnixNano()
		rs := status[0]
		if len(status) > 1 {
			i := t % int64(len(status))

			rs = status[i]

		}
		if responseJsonFlag {
			return c.Status(rs).JSON(fiber.Map{
				"status": rs,
			})
		} else {
			return c.SendString(fmt.Sprintf("status: %d\n", rs))
		}
	}

}
