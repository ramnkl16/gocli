package teams

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostMessage_OK(t *testing.T) {
	t.Parallel()
	var got MessagePayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type %q", ct)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := PostMessage(context.Background(), srv.URL, "hello"); err != nil {
		t.Fatal(err)
	}
	if got.Type != "message" {
		t.Fatalf("type %q", got.Type)
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("attachments len %d", len(got.Attachments))
	}
	attachment := got.Attachments[0]
	if attachment.ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Fatalf("content type %q", attachment.ContentType)
	}
	if attachment.Content.Type != "AdaptiveCard" {
		t.Fatalf("card type %q", attachment.Content.Type)
	}
	if len(attachment.Content.Body) != 1 {
		t.Fatalf("body len %d", len(attachment.Content.Body))
	}
	if attachment.Content.Body[0].Text != "hello" {
		t.Fatalf("text %q", attachment.Content.Body[0].Text)
	}
}

func TestPostMessage_EmptyURL(t *testing.T) {
	t.Parallel()
	err := PostMessage(context.Background(), "", "x")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestPostMessage_Non2xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	}))
	defer srv.Close()
	err := PostMessage(context.Background(), srv.URL, "hi")
	if err == nil {
		t.Fatal("want error")
	}
}
