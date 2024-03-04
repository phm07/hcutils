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
```