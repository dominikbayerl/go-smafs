# go-smafs

A FUSE filesystem implementation for the integrated logger of SMA solar inverters.

## Getting Started

```
go run main.go [-debug] <url> <mountpoint>
# example:
go run main.go -debug https://usr:secret@sma733147246.lan/ /mnt/smafs
```

## License

This project is licensed under the GPLv3 license - see the [LICENSE.md](LICENSE.md) file for details
