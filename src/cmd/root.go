package cmd

import (
	"hcutils/src/cmd/download"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "hcutils",
		Short: "Hetzner Cloud utilities",
	}
	rootCmd.AddCommand(
		download.NewDownloadCommand(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
