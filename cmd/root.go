package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	Port int
)

// Version is set at build time via -ldflags "-X github.com/xunull/nght/cmd.Version=..."
var Version = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "nght",
	Short:   "a gin/fiber web server for nginx http test",
	Long:    `a gin/fiber web server for nginx http test`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&Port, "port", "p", 8080, "the port")
	rootCmd.SetVersionTemplate("{{.Use}} version {{.Version}}\n")
}
