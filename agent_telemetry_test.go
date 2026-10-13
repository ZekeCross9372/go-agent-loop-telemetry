package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDiagnoseRecordsFailureAgainstSameKey(t *testing.T) {
	cases := []struct{ name, task string }{{name: "failed release", task: "compile release agent"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			client := &Client{Key: "test-key", HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				paths = append(paths, r.URL.Path)
				if r.URL.Path == "/v1/chat/completions" {
					return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error":{"message":"request rejected"}}`)), Header: make(http.Header)}, nil
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":{}}`)), Header: make(http.Header)}, nil
			})}}
			if _, err := diagnose(client, tc.task); err == nil {
				t.Fatal("expected rejected chat")
			}
			if strings.Join(paths, ",") != "/v1/chat/completions,/v1/errors/capture,/v1/metrics/report" {
				t.Fatalf("paths=%v", paths)
			}
		})
	}
}
