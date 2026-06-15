package cmd

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xunull/nght/internal/fiber_server"
	"github.com/xunull/nght/internal/gin_server"
	"github.com/xunull/nght/internal/global"
)

var (
	AppName      string
	ServerType   string
	ResponseJson bool
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start web server",
	Long:  `start web server`,
	Run: func(cmd *cobra.Command, args []string) {

		global.SetAppName(AppName)

		switch ServerType {
		case "gin":
			gin_server.Serve(Port)
		case "fiber":
			fiber_server.SetResponseJson(ResponseJson)
			adminToken := readAdminToken()
			fiber_server.Serve(Port, adminToken)
		default:
			log.Fatal("server type not support")
		}
	},
}

// readAdminToken returns NGHT_ADMIN_TOKEN from the env, or "" if unset.
// If the value contains any whitespace (space, tab, CR, LF), a warning
// is logged at startup because the middleware does exact-match with
// no trimming, so a stray space in the secret will silently fail all
// admin requests.
func readAdminToken() string {
	tok := os.Getenv("NGHT_ADMIN_TOKEN")
	if strings.ContainsAny(tok, " \t\n\r") {
		log.Printf("WARN: NGHT_ADMIN_TOKEN contains whitespace; the token is matched exactly (no trim) so a stray space in NGHT_ADMIN_TOKEN will silently fail admin auth. Verify the secret matches the X-Admin-Token header byte-for-byte.")
	}
	return tok
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().StringVarP(&ServerType, "type", "t", "gin", "server type")
	serverCmd.PersistentFlags().BoolVar(&ResponseJson, "response-json", false, "response json (fiber only)")
	serverCmd.PersistentFlags().StringVar(&AppName, "app-name", "nght", "app name")
}
