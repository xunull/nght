package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	RemoteHost string
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "start test client",
	Long:  `start test client`,
	Run: func(cmd *cobra.Command, args []string) {
		// todo
		fmt.Println("client called")
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVar(&RemoteHost, "host", "127.0.0.1", "remote server addr")
}
