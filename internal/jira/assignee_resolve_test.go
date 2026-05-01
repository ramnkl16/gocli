package jira

import (
	"strings"
	"testing"
)

func TestResolveAssignee(t *testing.T) {
	aliases := map[string]string{"bob": "bob@corp.example", " x ": " trimmed ", "bad": ""}
	got, err := ResolveAssignee(aliases, "BOB ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "bob@corp.example" {
		t.Fatalf("bob alias: got %q", got)
	}
	got, err = ResolveAssignee(aliases, "x")
	if err != nil || strings.TrimSpace(got) != "trimmed" {
		t.Fatalf("x alias: got %q err %v", got, err)
	}
	got, err = ResolveAssignee(aliases, "nobody@corp.example")
	if err != nil || got != "nobody@corp.example" {
		t.Fatalf("passthrough: got %q err %v", got, err)
	}
	_, err = ResolveAssignee(aliases, "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only assignee")
	}
	got, err = ResolveAssignee(nil, "plain")
	if err != nil || got != "plain" {
		t.Fatalf("nil aliases: got %q err %v", got, err)
	}
	_, err = ResolveAssignee(aliases, "bad")
	if err == nil {
		t.Fatal("expected error for empty alias target")
	}
}
