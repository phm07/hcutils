package upload

import (
	"context"
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"hcutils/src/util"
	"io"
	"os"
)

func NewUploadVolumeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Upload a volume",
		Long: `Upload a volume to the cloud from your local machine. For example:

hcutils upload volume --location fsn1 --size 10 --name my-uploaded-volume volume.tar.gz`,
		SilenceUsage: true,
		RunE:         uploadVolume,
		Args:         cobra.ExactArgs(1),
	}

	cmd.Flags().String("name", "", "Name of the new volume")

	cmd.Flags().String("location", "", "Location to create the volume in")
	_ = cmd.MarkFlagRequired("location")

	cmd.Flags().Int("size", 0, "Size of the volume in GB")
	_ = cmd.MarkFlagRequired("size")

	return cmd
}

func uploadVolume(cmd *cobra.Command, args []string) (err error) {

	location, _ := cmd.Flags().GetString("location")
	size, _ := cmd.Flags().GetInt("size")
	name, _ := cmd.Flags().GetString("name")

	ctx := context.Background()

	client, err := util.InitHCloudClient()
	if err != nil {
		return err
	}

	signer, pubKey, err := util.GenerateSSHKeypair()
	if err != nil {
		return err
	}

	sshKey, _, err := client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      "hcutil-temp-ssh-" + util.RandomDigits(5),
		PublicKey: pubKey,
		Labels:    map[string]string{"created-by": "hcutils"},
	})
	if err != nil {
		return err
	}

	fmt.Println("Creating temporary server...")
	result, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       "hcutil-temp-srv-" + util.RandomDigits(5),
		Location:   &hcloud.Location{Name: location},
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

	_, err = client.SSHKey.Delete(ctx, sshKey)
	if err != nil {
		return err
	}

	fmt.Println("Waiting for server to start...")
	err = util.WaitForServerStart(server)
	if err != nil {
		return err
	}

	if name == "" {
		name = "hcutil-uploaded-volume-" + util.RandomDigits(5)
	}

	fmt.Println("Creating volume...")
	volumeResult, _, err := client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Server:    server,
		Name:      name,
		Automount: hcloud.Ptr(true),
		Format:    hcloud.Ptr("ext4"),
		Size:      size,
	})
	if err != nil {
		return err
	}
	volume := volumeResult.Volume

	if err = util.WaitForAction(ctx, client, volumeResult.Action); err != nil {
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

	inFile, err := os.OpenFile(args[0], os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := inFile.Close()
		err = errors.Join(err, closeErr)
	}()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	fmt.Printf("Uploading %s to %s...\n", args[0], volume.Name)
	pipe, err := session.StdinPipe()
	if err != nil {
		return err
	}

	err = session.Start(fmt.Sprintf(`
DIR=/mnt/HC_Volume_%d/
while [ ! -d $DIR ]; do
	sleep 1
done
cd $DIR && tar xfz -
sync`, volume.ID))

	if err != nil {
		return err
	}

	_, err = io.Copy(pipe, inFile)
	if err != nil {
		return err
	}

	err = pipe.Close()
	if err != nil {
		return err
	}

	err = session.Wait()
	if err != nil {
		return err
	}

	return nil
}
