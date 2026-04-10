package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSharedSecretFromEnv(t *testing.T) {
	for _, tt := range []struct {
		description    string
		secretEnv      string
		secretFileEnv  string
		secretFileBody string
		wantSecret     string
		wantErr        bool
	}{
		{
			description: "reads secret from PS_SHARED_SECRET",
			secretEnv:   "mysecret",
			wantSecret:  "mysecret",
		},
		{
			description:    "reads secret from PS_SHARED_SECRET_FILE",
			secretFileBody: "mysecret",
			wantSecret:     "mysecret",
		},
		{
			description:    "trims trailing newline from secret file",
			secretFileBody: "mysecret\n",
			wantSecret:     "mysecret",
		},
		{
			description:    "trims trailing CRLF from secret file",
			secretFileBody: "mysecret\r\n",
			wantSecret:     "mysecret",
		},
		{
			description:    "PS_SHARED_SECRET_FILE takes precedence over PS_SHARED_SECRET",
			secretEnv:      "envSecret",
			secretFileBody: "fileSecret",
			wantSecret:     "fileSecret",
		},
		{
			description: "returns error when neither variable is set",
			wantErr:     true,
		},
		{
			description:   "returns error when PS_SHARED_SECRET_FILE points to nonexistent file",
			secretFileEnv: "/nonexistent/path/secret.txt",
			wantErr:       true,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			t.Setenv("PS_SHARED_SECRET", tt.secretEnv)

			if tt.secretFileBody != "" {
				f := filepath.Join(t.TempDir(), "secret.txt")
				if err := os.WriteFile(f, []byte(tt.secretFileBody), 0600); err != nil {
					t.Fatalf("failed to write secret file: %v", err)
				}
				t.Setenv("PS_SHARED_SECRET_FILE", f)
			} else {
				t.Setenv("PS_SHARED_SECRET_FILE", tt.secretFileEnv)
			}

			got, err := sharedSecretFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantSecret {
				t.Errorf("got=%q, want=%q", got, tt.wantSecret)
			}
		})
	}
}
