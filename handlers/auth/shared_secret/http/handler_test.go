package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpAuth "github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/http"
)

func TestStartSession(t *testing.T) {
	for _, tt := range []struct {
		description    string
		secretKey      string
		requestBody    string
		expectedStatus int
		expectedErr    error
	}{
		{
			description:    "accept valid credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": "mysecret"}`,
			expectedStatus: http.StatusOK,
			expectedErr:    nil,
		},
		{
			description:    "reject invalid credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": "wrongsecret"}`,
			expectedStatus: http.StatusUnauthorized,
			expectedErr:    httpAuth.ErrInvalidCredentials,
		},
		{
			description:    "reject empty credentials",
			secretKey:      "mysecret",
			requestBody:    `{"sharedSecretKey": ""}`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    httpAuth.ErrEmptyCredentials,
		},
		{
			description:    "reject malformed JSON",
			secretKey:      "mysecret",
			requestBody:    `{malformed`,
			expectedStatus: http.StatusBadRequest,
			expectedErr:    httpAuth.ErrMalformedRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			auth, err := httpAuth.New(tt.secretKey)
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

			body, _ := io.ReadAll(res.Body)
			if got, want := getError(res.StatusCode, strings.TrimSpace(string(body))), tt.expectedErr; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if tt.expectedErr != nil {
				return
			}

			cookie := getCookie(t, res)
			if got, want := cookie.Name, "sharedSecret"; got != want {
				t.Errorf("cookie name=%v, want=%v", got, want)
			}
			if !cookie.HttpOnly {
				t.Error("cookie is not HTTP-only")
			}
			if got, want := cookie.MaxAge, 30*24*60*60; got != want {
				t.Errorf("cookie MaxAge=%v, want=%v", got, want)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	for _, tt := range []struct {
		description string
		secretKey   string
		cookieVal   string
		want        bool
	}{
		{
			description: "accept valid cookie",
			secretKey:   "mysecret",
			cookieVal:   createValidCookie(t, "mysecret"),
			want:        true,
		},
		{
			description: "reject invalid cookie",
			secretKey:   "mysecret",
			cookieVal:   createValidCookie(t, "wrongsecret"),
			want:        false,
		},
		{
			description: "reject empty cookie",
			secretKey:   "mysecret",
			cookieVal:   "",
			want:        false,
		},
		{
			description: "reject malformed base64 cookie",
			secretKey:   "mysecret",
			cookieVal:   "not-base64!",
			want:        false,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			auth, err := httpAuth.New(tt.secretKey)
			if err != nil {
				t.Fatalf("failed to create authenticator: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookieVal != "" {
				req.AddCookie(&http.Cookie{
					Name:  "sharedSecret",
					Value: tt.cookieVal,
				})
			}

			if got, want := auth.Authenticate(req), tt.want; got != want {
				t.Errorf("got=%v, want=%v", got, want)
			}
		})
	}
}

func TestClearSession(t *testing.T) {
	auth, err := httpAuth.New("mysecret")
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

// Helper function to get error from status code and response body
func getError(statusCode int, body string) error {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return httpAuth.ErrInvalidCredentials
	case http.StatusBadRequest:
		switch body {
		case httpAuth.ErrEmptyCredentials.Error():
			return httpAuth.ErrEmptyCredentials
		default:
			return httpAuth.ErrMalformedRequest
		}
	default:
		return fmt.Errorf("unexpected status code: %d", statusCode)
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

// Helper function to create a valid cookie value for testing
func createValidCookie(t *testing.T, secret string) string {
	t.Helper()
	auth, err := httpAuth.New(secret)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth", createJSONBody(t, secret))
	auth.StartSession(w, req)

	return getCookie(t, w.Result()).Value
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
