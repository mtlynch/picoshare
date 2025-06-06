package shared_secret_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
)

func TestStartSession(t *testing.T) {
	for _, tt := range []struct {
		description    string
		secretKey      string
		requestBody    string
		expectedStatus int
	}{
		{
			description:    "accept valid credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": "mysecret"}`,
			expectedStatus: http.StatusOK,
		},
		{
			description:    "reject invalid credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": "wrongsecret"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			description:    "reject empty credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			description:    "reject malformed JSON",
			secretKey:      "mysecret",
			requestBody:    `{malformed`,
			expectedStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			auth, err := shared_secret.New(tt.secretKey)
			if err != nil {
				t.Fatalf("failed to create authenticator: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBufferString(tt.requestBody))
			w := httptest.NewRecorder()

			auth.StartSession(w, req)

			res := w.Result()

			if got, want := res.StatusCode, tt.expectedStatus; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			// Only check the response if the request succeeded.
			if res.StatusCode != http.StatusOK {
				return
			}

			cookie := getCookie(t, res)
			if got, want := cookie.Name, "sharedSecret"; got != want {
				t.Errorf("cookie name=%v, want=%v", got, want)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	secretKey := "mysecret"

	// Create authenticator.
	auth, err := shared_secret.New(secretKey)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// Start a valid session to get a valid cookie.
	w := httptest.NewRecorder()
	sessionReq := httptest.NewRequest(http.MethodPost, "/auth", createJSONBody(t, secretKey))
	auth.StartSession(w, sessionReq)

	validCookie := getCookie(t, w.Result())

	t.Run("valid cookie should authenticate successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(validCookie)
		if got, want := auth.Authenticate(req), true; got != want {
			t.Errorf("got=%v, want=%v", got, want)
		}
	})

	t.Run("request with no cookie should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if got, want := auth.Authenticate(req), false; got != want {
			t.Errorf("got=%v, want=%v", got, want)
		}
	})

	t.Run("empty cookie should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "sharedSecret",
			Value: "",
		})
		if got, want := auth.Authenticate(req), false; got != want {
			t.Errorf("got=%v, want=%v", got, want)
		}
	})

	t.Run("malformed base64 cookie should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "sharedSecret",
			Value: "not-base64!",
		})
		if got, want := auth.Authenticate(req), false; got != want {
			t.Errorf("got=%v, want=%v", got, want)
		}
	})

	t.Run("cookie created with wrong secret should fail", func(t *testing.T) {
		wrongAuth, err := shared_secret.New("wrongsecret")
		if err != nil {
			t.Fatalf("failed to create wrong authenticator: %v", err)
		}

		wrongW := httptest.NewRecorder()
		wrongReq := httptest.NewRequest(http.MethodPost, "/auth", createJSONBody(t, "wrongsecret"))
		wrongAuth.StartSession(wrongW, wrongReq)
		wrongCookie := getCookie(t, wrongW.Result())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(wrongCookie)
		if got, want := auth.Authenticate(req), false; got != want {
			t.Errorf("got=%v, want=%v", got, want)
		}
	})
}

func TestClearSession(t *testing.T) {
	auth, err := shared_secret.New("mysecret")
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	w := httptest.NewRecorder()
	auth.ClearSession(w)

	res := w.Result()
	cookie := getCookie(t, res)

	if got, want := cookie.Name, "sharedSecret"; got != want {
		t.Errorf("cookie name=%v, want=%v", got, want)
	}
	if got, want := cookie.Value, ""; got != want {
		t.Errorf("cookie value=%v, want=%v", got, want)
	}
	if !cookie.HttpOnly {
		t.Error("cookie is not HTTP-only")
	}
	if got, want := cookie.MaxAge, -1; got != want {
		t.Errorf("cookie MaxAge=%v, want=%v", got, want)
	}
}

// Helper function to get cookie from response
func getCookie(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	return cookies[0]
}

// Helper function to create a JSON request body
func createJSONBody(t *testing.T, secret string) *bytes.Buffer {
	t.Helper()
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{
		SharedSecretKey: secret,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}
	return &buf
}
