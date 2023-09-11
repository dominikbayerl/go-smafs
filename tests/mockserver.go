package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"path/filepath"
	"strings"

	"github.com/dominikbayerl/go-smafs/types"
)

type LoggingTransport struct{ http.Transport }

func (s *LoggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, []byte("\n\n")...)
	bytes = append(bytes, respBytes...)

	return resp, err
}

func NewMockServer(fsys fs.FS) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/dyn/login.json", func(w http.ResponseWriter, r *http.Request) {
		responseJSON := `{"result":{"sid":"test-sid"}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	})
	mux.HandleFunc("/dyn/logout.json", func(w http.ResponseWriter, r *http.Request) {
		responseJSON := `{"result":{"isLogin":false}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	})
	mux.HandleFunc("/dyn/getFS.json", func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Device []interface{} `json:"destDev"`
			Path   string        `json:"path"`
		}
		requestBody, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(requestBody, &requestPayload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requestPath := strings.Trim(requestPayload.Path, "/")
		if requestPath == "" {
			requestPath = "."
		}
		content, err := fs.ReadDir(fsys, requestPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading content: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		entries := make([]types.FSEntry, len(content))
		for idx, entry := range content {
			info, _ := entry.Info()
			if entry.IsDir() {
				entries[idx] = types.FSEntry{DirectoryName: entry.Name(), Timestamp: uint64(info.ModTime().Unix())}
			} else {
				entries[idx] = types.FSEntry{Filename: entry.Name(), Timestamp: uint64(info.ModTime().Unix()), Size: uint64(info.Size())}
			}
		}

		listing := make(map[string][]types.FSEntry)
		listing[requestPayload.Path] = entries
		devices := make(map[string]map[string][]types.FSEntry)
		devices["mockserver"] = listing
		responseJSON, err := json.Marshal(types.FSResponse{Devices: devices})
		if err != nil {
			http.Error(w, fmt.Sprintf("error marshaling JSON: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseJSON))
	})
	mux.HandleFunc("/fs/", func(w http.ResponseWriter, r *http.Request) {
		p, err := filepath.Rel("/fs/", r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("error invalid path: %v", err), http.StatusBadRequest)
			return
		}
		file, err := fsys.Open(p)
		if err != nil {
			http.Error(w, fmt.Sprintf("error opening file: %v", err), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading file: %v", err), http.StatusInternalServerError)
			return
		}
	})

	return httptest.NewServer(mux)
}
