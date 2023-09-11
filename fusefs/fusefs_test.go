package fusefs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"

	"github.com/dominikbayerl/go-smafs/sma"
	"github.com/dominikbayerl/go-smafs/tests"
	"github.com/hanwen/go-fuse/v2/fs"
)

// Define a mock HTTP server and SMAApi for testing
func setupTest(responseJSON string) (*httptest.Server, *sma.SMAApi) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the API response for GetFS
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, responseJSON)
	}))

	api := sma.SMAApi{Base: server.URL, Client: *http.DefaultClient}
	return server, &api
}

func TestMount(t *testing.T) {
	root := FuseNode{root: &FuseRoot{ctx: context.Background()}}
	opts := &fs.Options{}
	opts.Debug = true

	dir, err := os.MkdirTemp("", "fusefs-test")
	if err != nil {
		t.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	server, err := fs.Mount(dir, &root, opts)
	if err != nil {
		t.Errorf("error during fs mount: %v", err)
	}
	defer server.Unmount()

	server.WaitMount()
	// do some things.
}

func TestReaddir(t *testing.T) {
	m := fstest.MapFS{
		"DIAGNOSE/file1.txt": &fstest.MapFile{Data: []byte("file1.txt content\n")},
		"DIAGNOSE/file2.txt": &fstest.MapFile{Data: []byte("file2.txt content\n")},
		"SYSLOG/blarg":       &fstest.MapFile{Data: []byte("blarg content\n")},
	}
	mock := tests.NewMockServer(m)
	defer mock.Close()

	root := FuseNode{root: &FuseRoot{api: &sma.SMAApi{Base: mock.URL, Client: *http.DefaultClient}, ctx: context.WithValue(context.Background(), "sid", "test-sid")}}
	opts := &fs.Options{}
	opts.Debug = true

	dir, err := os.MkdirTemp("", "fusefs-test")
	if err != nil {
		t.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	server, err := fs.Mount(dir, &root, opts)
	if err != nil {
		t.Errorf("error during fs mount: %v", err)
	}
	defer server.Unmount()

	server.WaitMount()

	// do some things
	entries, err := os.ReadDir(dir + "/DIAGNOSE/")
	if err != nil {
		t.Errorf("error during readdir: %v", err)
	}
	t.Logf("dir entries: %v\n", entries)
}
