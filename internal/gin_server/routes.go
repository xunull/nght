package gin_server

func AddRoute() {
	// 请求是什么就返回什么
	ginServer.Any("/echo/:text", EchoTextResp)

	// 请求是什么状态码，返回什么状态码
	ginServer.Any("/status/:status", StatusResp)
	// 请求多少秒后返回
	ginServer.Any("/response_time/:time", ResponseTimeResp)
	// 随机返回状态码
	ginServer.Any("/random/:statusRandom", RandomStatusResp)
	ginServer.Any("/random_crash/:percentage/:statusRandom", RandomCrashResp)

	healthGroup := ginServer.Group("/health")
	{
		healthGroup.Any("", HealthResp)
		healthGroup.Any("/random/:percentage", HealthRandomResp)
		healthGroup.Any("/true", HealthTrueResp)
		healthGroup.Any("/false", HealthFalseResp)
	}
}
