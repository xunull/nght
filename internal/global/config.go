package global

import "os"

var (
	Hostname string
	AppName  string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	Hostname = hostname
}

func SetAppName(appName string) {
	AppName = appName
}
