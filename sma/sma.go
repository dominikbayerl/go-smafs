package sma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dominikbayerl/go-smafs/types"
)

// SMAApi is a FUSE filesystem that uses an HTTP API for file access.
type SMAApi struct {
	Base string
	// Runtime
	Client http.Client
}

// EnsureTrailingSlash ensures that a string has a trailing slash.
func EnsureTrailingSlash(input string) string {
	if !strings.HasSuffix(input, "/") {
		return input + "/"
	}
	return input
}

func (api *SMAApi) Login(profile, password string) (string, error) {
	loginURL := fmt.Sprintf("%s/dyn/login.json", api.Base)

	// Define the request payload as a struct
	requestPayload := struct {
		Right string `json:"right"`
		Pass  string `json:"pass"`
	}{
		Right: profile,
		Pass:  password,
	}

	// Convert the payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %v", err)
	}
	// Create a POST request
	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set request headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// Send the request using the client
	resp, err := api.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Define a struct for parsing the response JSON
	var response struct {
		Result struct {
			SID string `json:"sid"`
		} `json:"result"`
	}

	// Unmarshal the response JSON into the struct
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %v", err)
	}

	return response.Result.SID, nil
}

func (api *SMAApi) Logout(ctx context.Context) (bool, error) {
	// Define the URL for the Logout endpoint
	sid := ctx.Value("sid")
	url := fmt.Sprintf("%s/dyn/logout.json?sid=%s", api.Base, sid)

	// Define an empty payload for the request
	requestPayload := map[string]interface{}{}

	// Convert the payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return false, fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Create a POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}

	// Set request headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// Send the request using the client
	resp, err := api.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the response JSON into a LogoutResponse struct
	var logoutResponse struct {
		Result struct {
			IsLogin bool `json:"isLogin"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &logoutResponse); err != nil {
		return false, fmt.Errorf("error unmarshaling response JSON: %v", err)
	}

	return !logoutResponse.Result.IsLogin, nil
}

func (api *SMAApi) GetFS(ctx context.Context, path string) ([]types.FSEntry, error) {
	// Define the URL for the GetFS endpoint
	sid := ctx.Value("sid")
	url := fmt.Sprintf("%s/dyn/getFS.json?sid=%s", api.Base, sid)

	// Define the request payload
	requestPayload := map[string]interface{}{
		"destDev": []interface{}{},
		"path":    EnsureTrailingSlash(path),
	}

	// Convert the payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Create a POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set request headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// Send the request using the client
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the response JSON into an FSResponse struct
	var fsResponse types.FSResponse
	if err := json.Unmarshal(responseBody, &fsResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling response JSON: %v", err)
	}

	if len(fsResponse.Devices) != 1 {
		return nil, fmt.Errorf("error multiple devices not supported")
	}

	var device string
	var paths map[string][]types.FSEntry

	for k, v := range fsResponse.Devices {
		device = k
		paths = v
		break
	}
	_ = device // unused

	if len(paths) != 1 {
		return nil, fmt.Errorf("error multiple path responses not supported")
	}

	var respPath string
	var entries []types.FSEntry
	for k, v := range paths {
		respPath = k
		entries = v
	}

	if !(respPath == path || strings.TrimRight(respPath, "/") == path) {
		return nil, fmt.Errorf("error invalid response path. Expected: %v, Actual: %v", path, respPath)
	}

	return entries, nil
}

func (api *SMAApi) Download(ctx context.Context, filename string) ([]byte, error) {
	sid := ctx.Value("sid")
	url := fmt.Sprintf("%s/fs/%s?sid=%s", api.Base, filename, sid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}
	return responseBody, nil
}
