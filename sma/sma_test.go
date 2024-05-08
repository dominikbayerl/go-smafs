package sma

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dominikbayerl/go-smafs/types"
)

func TestLogin(t *testing.T) {
	// Create a mock HTTP server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the API response for successful login
		responseJSON := `{"result":{"sid":"test-sid"}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	// Create an API instance with the mock server's URL
	api := SMAApi{Base: server.URL, Client: *http.DefaultClient}

	// Call the Login method
	sid, err := api.Login("foo", "bar")
	if err != nil {
		t.Errorf("Login returned an error: %v", err)
	}

	// Check if the returned SID is as expected
	expectedSID := "test-sid"
	if sid != expectedSID {
		t.Errorf("Expected SID: %s, Got: %s", expectedSID, sid)
	}
}

func TestLogout(t *testing.T) {
	// Create a mock HTTP server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the API response for Logout
		responseJSON := `{"result":{"isLogin":false}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	// Create an API instance with the mock server's URL
	client := SMAApi{Base: server.URL, Client: *http.DefaultClient}
	ctx := context.Background()

	// Call the Logout method
	logoutResult, err := client.Logout(context.WithValue(ctx, types.ApiContextKey("sid"), "test-sid"))
	if err != nil {
		t.Errorf("Logout returned an error: %v", err)
	}

	// Check the result
	expectedResult := true // Expecting a successful logout, which sets IsLogin to false

	if logoutResult != expectedResult {
		t.Errorf("Logout result does not match expected result. Expected: %v, Actual: %v", expectedResult, logoutResult)
	}
}

// Define a mock HTTP server and SMAApi for testing
func setupTest(responseJSON string) (*httptest.Server, *SMAApi) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the API response for GetFS
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, responseJSON)
	}))

	api := SMAApi{Base: server.URL, Client: *http.DefaultClient}
	return server, &api
}

func TestGetFS(t *testing.T) {
	responseJSON := `{
		"result": {
			"device1": {
				"/DIAGNOSE/": [
					{"f": "file1.txt", "tm": 1684094403, "s": 1024},
					{"d": "directory1", "tm": 1684094407},
					{"f": "file2.txt", "tm": 1694580920, "s": 2048}
				]
			}
		}
	}`
	server, api := setupTest(responseJSON)
	defer server.Close()

	ctx := context.WithValue(context.Background(), types.ApiContextKey("sid"), "test-sid")
	path := "/DIAGNOSE/"

	entries, err := api.GetFS(ctx, path)
	if err != nil {
		t.Errorf("GetFS returned an error: %v", err)
	}

	// Check the number of entries
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, but got %d", len(entries))
	}

	// Check the first entry (file)
	if entries[0].Filename != "file1.txt" || entries[0].DirectoryName != "" || entries[0].Timestamp != 1684094403 || entries[0].Size != 1024 {
		t.Errorf("Invalid entry[0]: %+v", entries[0])
	}

	// Check the second entry (directory)
	if entries[1].Filename != "" || entries[1].DirectoryName != "directory1" || entries[1].Timestamp != 1684094407 || entries[1].Size != 0 {
		t.Errorf("Invalid entry[1]: %+v", entries[1])
	}

	// Check the third entry (file)
	if entries[2].Filename != "file2.txt" || entries[2].DirectoryName != "" || entries[2].Timestamp != 1694580920 || entries[2].Size != 2048 {
		t.Errorf("Invalid entry[2]: %+v", entries[2])
	}
}

func TestGetFS_InvalidResponse(t *testing.T) {
	responseJSON := `{
		"result": {
			"device1": {
				"/DIAGNOSE/": [
					{"f": "file1.txt", "tm": 1684094403, "s": 1024},
					{"d": "directory1", "tm": 1684094407},
					{"f": "file2.txt", "tm": 1694580920, "s": 2048}
				]
			}
		}
	}`
	server, api := setupTest(responseJSON)
	defer server.Close()

	ctx := context.WithValue(context.Background(), types.ApiContextKey("sid"), "test-sid")
	path := "/INVALID/"

	_, err := api.GetFS(ctx, path)
	if err == nil {
		t.Error("Expected an error for an invalid response path, but got none")
	}
}

func TestGetFS_MultipleDevices(t *testing.T) {
	responseJSON := `{
		"result": {
			"device1": {
				"/DIAGNOSE/": [
					{"f": "file1.txt", "tm": 1684094403, "s": 1024}
				]
			},
			"device2": {
				"/DIAGNOSE/": [
					{"f": "file2.txt", "tm": 1694580920, "s": 2048}
				]
			}
		}
	}`
	server, api := setupTest(responseJSON)

	ctx := context.WithValue(context.Background(), types.ApiContextKey("sid"), "test-sid")
	path := "/DIAGNOSE/"

	_, err := api.GetFS(ctx, path)
	if err == nil {
		t.Error("Expected an error for multiple devices, but got none")
	}
	server.Close()
}

func TestGetFS_MultiplePaths(t *testing.T) {
	responseJSON := `{
		"result": {
			"device1": {
				"/DIAGNOSE/": [
					{"f": "file1.txt", "tm": 1684094403, "s": 1024},
					{"d": "directory1", "tm": 1684094407}
				],
				"/INVALID/": [
					{"f": "file2.txt", "tm": 1694580920, "s": 2048}
				]
			}
		}
	}`
	server, api := setupTest(responseJSON)

	ctx := context.WithValue(context.Background(), types.ApiContextKey("sid"), "test-sid")
	path := "/DIAGNOSE/"

	_, err := api.GetFS(ctx, path)
	if err == nil {
		t.Error("Expected an error for multiple paths, but got none")
	}
	server.Close()
}
