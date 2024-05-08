package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/hanwen/go-fuse/v2/fs"

	"github.com/dominikbayerl/go-smafs/fusefs"
	"github.com/dominikbayerl/go-smafs/sma"
	"github.com/dominikbayerl/go-smafs/types"
)

type SMAFs struct {
	fs.Inode
}

func main() {
	debug := flag.Bool("debug", false, "print debugging messages.")
	insecure := flag.Bool("insecure", false, "skip TLS certificate verification")
	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	u, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatalf("error invalid url: %v\n", err)
	}

	if *insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Read profile and password from environment variables
	usernameFile := os.Getenv("SMAFS_USER")
	if usernameFile == "" {
		log.Fatal("SMAFS_USER environment variable not set")
	}
	username, err := os.ReadFile(usernameFile)
	if err != nil {
		log.Fatalf("error reading username from file: %v\n", err)
	}

	passwordFile := os.Getenv("SMAFS_PASS")
	if passwordFile == "" {
		log.Fatal("SMAFS_PASS environment variable not set")
	}

	password, err := os.ReadFile(passwordFile)
	if err != nil {
		log.Fatalf("error reading password from file: %v\n", err)
	}

	api := sma.SMAApi{Base: fmt.Sprintf("%s://%s", u.Scheme, u.Host), Client: *http.DefaultClient}
	sid, err := api.Login(string(username), string(password))
	if err != nil || sid == "" {
		log.Fatalf("error requesting session: %v\n", err)
	}
	ctx := context.WithValue(context.Background(), types.ApiContextKey("sid"), sid)
	defer api.Logout(ctx)

	root := fusefs.NewFuseFS(ctx, &api)
	opts := &fs.Options{}
	opts.Debug = *debug
	server, err := fs.Mount(flag.Arg(1), root, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	server.Wait()
}
