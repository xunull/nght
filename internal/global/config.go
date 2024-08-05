package global

import "os"

var (
	Hostname string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	Hostname = hostname
}
