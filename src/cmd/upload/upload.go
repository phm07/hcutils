package upload

import (
	"github.com/spf13/cobra"
)

func NewUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload resources to the cloud",
		Long:  `Upload resources like volumes and snapshots to the cloud.`,
	}
	cmd.AddCommand(NewUploadVolumeCommand())
	return cmd
}
