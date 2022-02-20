package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/memory"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestDeleteFile(t *testing.T) {
	memStore := memory.New()
	memStore.InsertEntry(types.UploadEntry{
		UploadMetadata: types.UploadMetadata{
			ID: types.EntryID("hR87apiUCjTV9E"),
		},
		Data: []byte("dummy data"),
	})

	s := handlers.New(mockAuthenticator{}, memStore)

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCjTV9E", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	_, err = memStore.GetEntry(types.EntryID("hR87apiUCjTV9E"))
	if _, ok := err.(store.EntryNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted", types.EntryID("hR87apiUCjTV9E"))
	}
}
