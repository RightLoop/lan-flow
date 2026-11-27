package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestItemsRevalidation(t *testing.T) {
	cfg := Config{DataDir: t.TempDir(), RetentionHours: 24}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(cfg, store, RuntimeInfo{})
	getItems := func(validator string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
		req.Header.Set("If-None-Match", validator)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	initial := getItems("")
	tag := initial.Header().Get("ETag")
	if initial.Code != http.StatusOK || tag == "" || initial.Body.String() != "{\"files\":null,\"messages\":null}\n" {
		t.Fatalf("initial response: status %d, ETag %q, body %q", initial.Code, tag, initial.Body)
	}
	for _, validator := range []string{tag, "*", `"other,tag", W/` + tag} {
		unchanged := getItems(validator)
		if unchanged.Code != http.StatusNotModified || unchanged.Body.Len() != 0 || unchanged.Header().Get("ETag") != tag {
			t.Fatalf("response to %q: status %d, ETag %q, body %q", validator, unchanged.Code, unchanged.Header().Get("ETag"), unchanged.Body)
		}
	}
	older, err := store.AddMessage("older")
	if err != nil {
		t.Fatal(err)
	}
	message, err := store.AddMessage("hello")
	if err != nil {
		t.Fatal(err)
	}
	changed := getItems(tag)
	newTag := changed.Header().Get("ETag")
	if changed.Code != http.StatusOK || newTag == tag || newTag == "" || !strings.Contains(changed.Body.String(), message.ID) {
		t.Fatalf("updated response: status %d, ETag %q, body %q", changed.Code, newTag, changed.Body)
	}
	snapshot, _ := store.itemsSnapshot()
	if found, err := store.DeleteMessage(message.ID); err != nil || !found {
		t.Fatalf("delete message: found=%v, error=%v", found, err)
	}
	if snapshot.Messages[0].ID != message.ID {
		t.Fatal("deletion changed a previously acquired snapshot")
	}
	deleted := getItems(newTag)
	if deleted.Code != http.StatusOK || strings.Contains(deleted.Body.String(), message.ID) || !strings.Contains(deleted.Body.String(), older.ID) {
		t.Fatalf("response after deletion: status %d, body %q", deleted.Code, deleted.Body)
	}
}

func TestCleanupSkipsUnchangedMetadata(t *testing.T) {
	cfg := Config{DataDir: t.TempDir(), RetentionHours: 24}
	filesPath := filepath.Join(cfg.DataDir, "metadata.json")
	messagesPath := filepath.Join(cfg.DataDir, "messages.json")
	// Existing metadata may contain both expired and active items in any order.
	legacy := []MessageItem{
		{ID: "old", Text: "expired", CreatedAt: time.Now().Add(-48 * time.Hour)},
		{ID: "new", Text: "active", CreatedAt: time.Now()},
	}
	if err := writeJSONAtomic(messagesPath, legacy); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, originalTag := store.itemsSnapshot()
	if err := store.CleanupExpired(); err != nil {
		t.Fatal(err)
	}
	items, currentTag := store.itemsSnapshot()
	if currentTag == originalTag || len(items.Messages) != 1 || items.Messages[0].ID != "new" {
		t.Fatalf("cleanup snapshot: ETag %q, messages %v", currentTag, items.Messages)
	}
	var persisted []MessageItem
	if err := readJSONIfExists(messagesPath, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 || persisted[0].ID != "new" {
		t.Fatalf("cleanup persisted messages: %v", persisted)
	}
	if _, err := os.Stat(filesPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup touched unchanged file metadata: %v", err)
	}
	// An attempted atomic rewrite would fail while its temporary path is a directory.
	if err := os.Mkdir(messagesPath+".tmp", 0755); err != nil {
		t.Fatal(err)
	}
	if err := store.CleanupExpired(); err != nil {
		t.Fatalf("cleanup attempted to write unchanged metadata: %v", err)
	}
	if _, err := store.AddMessage("unpersistable"); err == nil {
		t.Fatal("message should fail while its metadata cannot be persisted")
	}
	items, unchangedTag := store.itemsSnapshot()
	if unchangedTag != currentTag || len(items.Messages) != 1 {
		t.Fatalf("failed write changed snapshot: ETag %q, messages %v", unchangedTag, items.Messages)
	}
}
