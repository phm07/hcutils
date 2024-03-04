package download

import (
	"context"
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"hcutils/src/util"
	util3 "hcutils/src/util"
	"net"
	"os"
	"time"
)

type downloadType string

const (
	downloadTypeArchive downloadType = "archive"
	downloadTypeImage   downloadType = "image"
)

func NewDownloadVolumeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Download a volume",
		Long: `Download a volume from the cloud to the local machine. For example:

hcutils download volume --id 1234 --out volume.tar.gz`,
		SilenceUsage: true,
		RunE:         downloadVolume,
	}

	cmd.Flags().String("type", "archive", "Type of download (archive or image)")

	cmd.Flags().String("out", "", "Output file")

	cmd.Flags().String("id", "", "ID or name of the volume to download")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func downloadVolume(cmd *cobra.Command, _ []string) (err error) {

	ctx := context.Background()
	volumeId, _ := cmd.Flags().GetString("id")
	outFileName, _ := cmd.Flags().GetString("out")
	downloadTypeStr, _ := cmd.Flags().GetString("type")

	var dlType downloadType
	switch downloadTypeStr {
	case string(downloadTypeArchive), string(downloadTypeImage):
		dlType = downloadType(downloadTypeStr)
	default:
		return fmt.Errorf("invalid download type: %s", downloadTypeStr)
	}

	client, err := util3.InitHCloudClient()
	if err != nil {
		return err
	}

	volume, _, err := client.Volume.Get(cmd.Context(), volumeId)
	if err != nil {
		return err
	}

	if volume == nil {
		return fmt.Errorf("volume not found: %s", volumeId)
	}

	var reattachTo *hcloud.Server

	if volume.Server != nil {
		reattachTo, _, err = client.Server.GetByID(ctx, volume.Server.ID)
		if err != nil {
			return err
		}

		unmount, err := util3.AskConfirmation(
			fmt.Sprintf(
				"Volume %s is attached to server %s. To download the volume, it needs to be datached. "+
					"Do you want to unmount it?\nWarning: This could possibly lead to data loss",
				volume.Name, reattachTo.Name,
			),
		)
		if err != nil {
			return err
		}

		if !unmount {
			fmt.Println("Canceling")
			return nil
		}

		fmt.Printf("Detaching volume %s\n", volume.Name)
		action, _, err := client.Volume.Detach(ctx, volume)
		if err != nil {
			return err
		}

		if err = util.WaitForAction(ctx, client, action); err != nil {
			return err
		}
	}

	signer, pubKey, err := util3.GenerateSSHKeypair()
	if err != nil {
		return err
	}

	sshKey, _, err := client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      "hcutil-temp-ssh-" + util3.RandomDigits(5),
		PublicKey: pubKey,
		Labels:    map[string]string{"created-by": "hcutils"},
	})
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("Deleting SSH key...")
		_, deleteErr := client.SSHKey.Delete(ctx, sshKey)
		err = errors.Join(err, deleteErr)
	}()

	fmt.Println("Creating temporary server...")
	result, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       "hcutil-temp-srv-" + util3.RandomDigits(5),
		Location:   volume.Location,
		ServerType: &hcloud.ServerType{Name: "cx11"},
		Image:      &hcloud.Image{Name: "ubuntu-22.04"},
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		Labels:     map[string]string{"created-by": "hcutils"},
	})

	if err != nil {
		return err
	}
	server := result.Server

	defer func() {
		fmt.Println("Deleting temporary server...")
		res, _, deleteErr := client.Server.DeleteWithResult(ctx, server)
		err = errors.Join(err, deleteErr)
		if res != nil {
			err = errors.Join(err, util.WaitForAction(ctx, client, res.Action))
		}
	}()

	fmt.Println("Waiting server to start...")
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if time.Since(start) > time.Minute {
			return errors.New("timeout waiting for server to start")
		}
		conn, _ := net.DialTimeout("tcp", server.PublicNet.IPv4.IP.String()+":22", time.Second)
		if conn != nil {
			_ = conn.Close()
			break
		}
	}

	fmt.Println("Attaching volume...")
	action, _, err := client.Volume.AttachWithOpts(ctx, volume, hcloud.VolumeAttachOpts{
		Server:    server,
		Automount: hcloud.Ptr(true),
	})
	if err != nil {
		return err
	}

	if err = util.WaitForAction(ctx, client, action); err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClient, err := ssh.Dial("tcp", server.PublicNet.IPv4.IP.String()+":22", config)
	if err != nil {
		return err
	}
	defer func() {
		_ = sshClient.Close()
	}()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	if outFileName == "" {
		if dlType == downloadTypeArchive {
			outFileName = fmt.Sprintf("volume-%d.tar.gz", volume.ID)
		} else {
			outFileName = fmt.Sprintf("volume-%d.img.gz", volume.ID)
		}
	}

	outFile, err := os.OpenFile(outFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := outFile.Close()
		err = errors.Join(err, closeErr)
	}()

	session.Stdout = outFile

	fmt.Printf("Downloading volume %s to %s...\n", volume.Name, outFileName)
	if dlType == downloadTypeArchive {
		err = session.Run(fmt.Sprintf(`cd /mnt/HC_Volume_%d/ && tar czf - . | cat`, volume.ID))
	} else {
		err = session.Run(fmt.Sprintf(`dd if=%s bs=32M | gzip -f`, volume.LinuxDevice))
	}
	if err != nil {
		return err
	}

	if reattachTo != nil {
		reattach, err := util3.AskConfirmation(
			fmt.Sprintf(
				"Volume %s was attached to server %s. Do you want to reattach it?",
				volume.Name, reattachTo.Name,
			),
		)
		if err != nil {
			return err
		}
		if reattach {

			fmt.Printf("Reattaching volume %s to server %s\n", volume.Name, reattachTo.Name)
			action, _, err = client.Volume.Detach(ctx, volume)
			if err != nil {
				return err
			}
			if err = util.WaitForAction(ctx, client, action); err != nil {
				return err
			}

			action, _, err = client.Volume.Attach(ctx, volume, reattachTo)
			if err != nil {
				return err
			}
			if err = util.WaitForAction(ctx, client, action); err != nil {
				return err
			}
		}
	}

	return nil
}
