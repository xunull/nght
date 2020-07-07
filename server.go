package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	ginServer  *gin.Engine
	httpServer *http.Server
	mu         sync.Mutex
	health     = true
)

type StatusParam struct {
	Status int
}

// 请求是什么状态码,就返回什么状态码
func StatusResp(c *gin.Context) {
	var param StatusParam
	if err := c.ShouldBindUri(&param); err != nil {
		c.JSON(400, gin.H{"msg": err})
	} else {
		c.JSON(param.Status, gin.H{"status": param.Status})
	}
}

type ResponseTimeParam struct {
	Time int
}

// 请求是多长时间,就多长时间返回
func ResponseTimeResp(c *gin.Context) {
	var param ResponseTimeParam
	if err := c.ShouldBindUri(&param); err != nil {
		c.JSON(400, gin.H{"msg": err})
	} else {
		time.Sleep(time.Duration(param.Time) * time.Second)
		c.JSON(http.StatusOK, gin.H{"time": param.Time})
	}

}

func SplitStatus(target string) ([]int, error) {
	l := len(target)

	if l%3 != 0 {
		return nil, errors.New("请求参数不合法,每三位为一个状态码")
	} else {
		status := make([]int, l/3)
		for i := 0; i <= l-3; i += 3 {
			if s, err := strconv.Atoi(target[i : i+3]); err != nil {
				return nil, errors.New("请求参数不合法,请输入有效状态码")
			} else {
				status = append(status, s)
			}
		}
		return status, nil
	}

}

type RandomStatusParam struct {
	StatusRandom string
}

// 在给定的状态码见随机的返回,请求参数要合理
// random 为每次请求的时间种子
func RandomStatusResp(c *gin.Context) {
	var param RandomStatusParam
	if err := c.ShouldBindUri(&param); err != nil {
		c.JSON(400, gin.H{"msg": err})
	} else {
		l := len(param.StatusRandom)
		if status, err := SplitStatus(param.StatusRandom); err != nil {
			c.JSON(400, gin.H{"msg": err})
		} else {
			rand.Seed(time.Now().UnixNano())
			i := rand.Intn(l/3 - 1)
			c.JSON(status[i], gin.H{"msg": i})
		}
	}
}

type RandomCrashParam struct {
	Percentage   int
	StatusRandom string
}

// percentage 为200状态码的比例,当非200时,将从提供的状态码中随机返回一个
func RandomCrashResp(c *gin.Context) {
	var param RandomCrashParam
	if err := c.ShouldBindUri(&param); err != nil {
		c.JSON(400, gin.H{"msg": err})
	} else {
		l := len(param.StatusRandom)
		if status, err := SplitStatus(param.StatusRandom); err != nil {
			c.JSON(400, gin.H{"msg": err})
		} else {
			rand.Seed(time.Now().UnixNano())
			if rand.Intn(100) < param.Percentage {
				c.Status(http.StatusOK)
			} else {
				i := rand.Intn(l/3 - 1)
				c.JSON(status[i], gin.H{"msg": i})
			}
		}
	}
}

func HealthResp(c *gin.Context) {
	if health {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusBadGateway)
	}

}

type HealthRandomParam struct {
	Percentage int
}

// 随机返回502
func HealthRandomResp(c *gin.Context) {
	var param HealthRandomParam
	if err := c.ShouldBindUri(&param); err != nil {
		c.JSON(400, gin.H{"msg": err})
	} else {
		rand.Seed(time.Now().UnixNano())
		if rand.Intn(100) < param.Percentage {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusBadGateway)
		}
	}
}

// 健康检测返回健康
func HealthTrueResp(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()
	health = true
	c.Status(http.StatusOK)
}

// 健康检测返回不健康
func HealthFalseResp(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()
	health = false
	c.Status(http.StatusOK)
}

func AddRoute() {
	ginServer.Any("/status/:status", StatusResp)
	ginServer.Any("/response_time/:time", ResponseTimeResp)
	ginServer.Any("/random/:statusRandom", RandomStatusResp)
	ginServer.Any("/random_crash/:percentage/:statusRandom", RandomCrashResp)

	healthGroup := ginServer.Group("/health")
	{
		healthGroup.Any("/", HealthResp)
		healthGroup.Any("/random/:percentage", HealthRandomResp)
		healthGroup.Any("/true", HealthTrueResp)
		healthGroup.Any("/false", HealthFalseResp)
	}
}
