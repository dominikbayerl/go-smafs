package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/hanwen/go-fuse/v2/fs"

	"github.com/dominikbayerl/go-smafs/fusefs"
	"github.com/dominikbayerl/go-smafs/sma"
)

type SMAFs struct {
	fs.Inode
}

func main() {
	debug := flag.Bool("debug", false, "print debugging messages.")
	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatal("Usage: smafs <url> <mountpoint>")
	}

	u, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatalf("error invalid url: %v\n", err)
	}

	if *debug {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	api := sma.SMAApi{Base: fmt.Sprintf("%s://%s", u.Scheme, u.Host), Client: *http.DefaultClient}
	profile := u.User.Username()
	password, _ := u.User.Password()

	sid, err := api.Login(profile, password)
	if err != nil || sid == "" {
		log.Fatalf("error requesting session: %v\n", err)
	}
	ctx := context.WithValue(context.Background(), "sid", sid)
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
