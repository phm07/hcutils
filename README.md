# hcutils
hcutils is a CLI tool with a collection of utilities for Hetzner Cloud.

## Building

Using make:

```bash
make build
```

## Usage

Set the `HCLOUD_TOKEN` environment variable to your Hetzner Cloud API token.

### Examples

```bash
# Download volume 123 to volume.tar.gz
hcutils download volume --id 123 --out volume.tar.gz

# Download volume 123 as gzipped binary to volume.img.gz
hcutils download volume --id 123 --type image --out volume.img.gz

# Upload volume.tar.gz to volume with name my-volume and size of 10GB in location fsn1
hcutils upload volume --location fsn1 --size 10 --name my-volume volume.tar.gz
```