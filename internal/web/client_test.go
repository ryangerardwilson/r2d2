package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendsTheKeyAndHidesIt(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		if r.URL.Path == "/api/v1/posts" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"slug":"2026-11-21-quiet","title":"Quiet","body":"End."}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"api key required","token":"` + r.Header.Get("Authorization") + `"}`))
	}))
	defer srv.Close()
	client := &Client{Origin: srv.URL, Token: "rgw_secret"}
	post, err := client.CreatePost("Quiet", "End.")
	if err != nil || post.Slug != "2026-11-21-quiet" {
		t.Fatalf("post %#v %v", post, err)
	}
	if gotAuth != "Bearer rgw_secret" {
		t.Fatal("missing bearer")
	}
	if strings.Contains(gotBody, "slug") {
		t.Fatalf("create sent a slug: %s", gotBody)
	}
	_, err = client.ListPosts()
	if err == nil || strings.Contains(err.Error(), "rgw_secret") || strings.Contains(err.Error(), "Bearer") {
		t.Fatalf("error leaked the key: %v", err)
	}
}

func TestDeleteTreatsNoContentAsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client := &Client{Origin: srv.URL, Token: "rgw_secret"}
	if err := client.DeletePost("one"); err != nil {
		t.Fatal(err)
	}
}
