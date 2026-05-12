package httperrorreturn_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mtlynch/picoshare/build/httperrorreturn"
)

func TestCheckPaths(t *testing.T) {
	t.Run("reports http error that falls through to later code", func(t *testing.T) {
		sourcePath := writeTempGoFile(t, `package example

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
	}

	w.WriteHeader(http.StatusNoContent)
}`)

		issues, err := httperrorreturn.CheckPaths(sourcePath)
		if got, want := err, error(nil); got != want {
			t.Fatalf("err=%v, want=%v", got, want)
		}
		if got, want := len(issues), 1; got != want {
			t.Fatalf("len(issues)=%d, want=%d", got, want)
		}
		if got, want := issues[0].Line, 7; got != want {
			t.Errorf("issues[0].Line=%d, want=%d", got, want)
		}
	})

	t.Run("allows enclosing return after if else branches", func(t *testing.T) {
		sourcePath := writeTempGoFile(t, `package example

import "net/http"

func handler(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
	} else {
		http.Error(w, "invalid request", http.StatusBadRequest)
	}
	return
}`)

		issues, err := httperrorreturn.CheckPaths(sourcePath)
		if got, want := err, error(nil); got != want {
			t.Fatalf("err=%v, want=%v", got, want)
		}
		if got, want := len(issues), 0; got != want {
			t.Fatalf("len(issues)=%d, want=%d", got, want)
		}
	})

	t.Run("allows http error at function end", func(t *testing.T) {
		sourcePath := writeTempGoFile(t, `package example

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
	}
}`)

		issues, err := httperrorreturn.CheckPaths(sourcePath)
		if got, want := err, error(nil); got != want {
			t.Fatalf("err=%v, want=%v", got, want)
		}
		if got, want := len(issues), 0; got != want {
			t.Fatalf("len(issues)=%d, want=%d", got, want)
		}
	})
}

func writeTempGoFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "input.go")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}

	return path
}
