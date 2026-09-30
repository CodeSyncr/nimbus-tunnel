package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CodeSyncr/nimbus/tunnel"
)

func TestAuthorizerNamesCloudflareBlock(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("Cf-Mitigated", "challenge")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html>Just a moment...</html>"))
	}))
	defer srv.Close()
	_, err := cloudAuthorizer(srv.URL)(context.Background(), "nb_cli_x", "")
	if err == nil || errors.Is(err, tunnel.ErrUnauthorized) || !strings.Contains(err.Error(), "Cloudflare blocked") {
		t.Fatalf("got %v", err)
	}
	var ae *tunnel.AuthError
	if errors.As(err, &ae) {
		t.Fatal("a CDN block must not read as a verdict on the token")
	}
	if !strings.HasPrefix(ua, "nimbus-tunnel/") {
		t.Fatalf("user agent %q", ua)
	}
}

func TestAuthorizerPassesCloudVerdicts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid or expired token"}`))
	}))
	defer srv.Close()
	_, err := cloudAuthorizer(srv.URL)(context.Background(), "nb_cli_x", "")
	var ae *tunnel.AuthError
	if !errors.As(err, &ae) || ae.Status != 401 {
		t.Fatalf("got %v", err)
	}
}
