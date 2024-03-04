package download

import (
	"github.com/spf13/cobra"
)

func NewDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download resources from the cloud",
		Long:  `Download resources like volumes and snapshots from the cloud.`,
	}
	cmd.AddCommand(NewDownloadVolumeCommand())
	return cmd
}
