package testutil

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v2/cassette"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

// NewVCRRecorder creates a new VCR recorder for testing
func NewVCRRecorder(t *testing.T, cassetteName string) (*recorder.Recorder, func()) {
	t.Helper()

	mode := recorder.ModeReplaying
	if os.Getenv("VCR_MODE") == "record" {
		mode = recorder.ModeRecording
	}

	cassettePath := filepath.Join("testdata", "fixtures", cassetteName)

	r, err := recorder.NewAsMode(cassettePath, mode, nil)
	if err != nil {
		t.Fatalf("Failed to create VCR recorder: %v", err)
	}

	// Don't match on request body for simplicity
	r.SetMatcher(func(r *http.Request, i cassette.Request) bool {
		return r.Method == i.Method && r.URL.String() == i.URL
	})

	// Cleanup function
	cleanup := func() {
		if err := r.Stop(); err != nil {
			t.Errorf("Failed to stop VCR recorder: %v", err)
		}
	}

	return r, cleanup
}

// VCRHTTPClient returns an HTTP client configured to use the VCR recorder
func VCRHTTPClient(r *recorder.Recorder) *http.Client {
	return &http.Client{
		Transport: r,
	}
}
