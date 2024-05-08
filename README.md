# go-smafs
![Go Package](https://github.com/dominikbayerl/go-smafs/actions/workflows/go.yml/badge.svg)

A FUSE filesystem implementation for the integrated logger of SMA solar inverters.

## Getting Started
To run the go-smafs daemon, you need to provide the URL of your SMA inverter and a mountpoint for the FUSE filesystem. Optionally, you can set the following environment variables:

- `SMAFS_USER`: The username for the SMA inverter (also called "Profile")
- `SMAFS_PASS`: The password for the SMA inverter

```
go run main.go [-debug] [-insecure ]<url> <mountpoint>
# example:
go run main.go -debug https://sma733147246.lan/ /mnt/smafs
```

## License

This project is licensed under the GPLv3 license - see the [LICENSE.md](LICENSE.md) file for details
