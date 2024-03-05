package cmd

import (
	"hcutils/src/cmd/download"
	"hcutils/src/cmd/upload"
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
		upload.NewUploadCommand(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
