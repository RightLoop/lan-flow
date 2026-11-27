package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
	StoredAs  string    `json:"storedAs"`
}

type MessageItem struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type ItemsResponse struct {
	Files    []FileItem    `json:"files"`
	Messages []MessageItem `json:"messages"`
}

type Store struct {
	cfg          Config
	mu           sync.RWMutex
	filesPath    string
	messagesPath string
	uploadDir    string
	files        []FileItem
	messages     []MessageItem
	etagPrefix   string
	revision     uint64
	itemsETag    string
}

func NewStore(cfg Config) (*Store, error) {
	generation, err := newID()
	if err != nil {
		return nil, err
	}
	s := &Store{
		cfg:          cfg,
		filesPath:    filepath.Join(cfg.DataDir, "metadata.json"),
		messagesPath: filepath.Join(cfg.DataDir, "messages.json"),
		uploadDir:    filepath.Join(cfg.DataDir, "files"),
		etagPrefix:   `"` + generation + "-",
	}
	s.advanceItemsETag()
	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	if err := readJSONIfExists(s.filesPath, &s.files); err != nil {
		return err
	}
	if err := readJSONIfExists(s.messagesPath, &s.messages); err != nil {
		return err
	}
	return nil
}

func readJSONIfExists(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// itemsSnapshot shares the store's immutable lists. Writers always publish new
// slices so a response can encode them after the read lock has been released.
func (s *Store) itemsSnapshot() (ItemsResponse, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items ItemsResponse
	if len(s.files) != 0 {
		items.Files = s.files
	}
	if len(s.messages) != 0 {
		items.Messages = s.messages
	}
	return items, s.itemsETag
}

func (s *Store) advanceItemsETag() {
	s.revision++
	s.itemsETag = s.etagPrefix + strconv.FormatUint(s.revision, 16) + `"`
}

func (s *Store) AddFile(originalName string, src io.Reader) (FileItem, error) {
	id, err := newID()
	if err != nil {
		return FileItem{}, err
	}
	name := cleanFileName(originalName)
	storedAs := id + "_" + name
	dstPath := filepath.Join(s.uploadDir, storedAs)

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return FileItem{}, err
	}
	size, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(dstPath)
		return FileItem{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dstPath)
		return FileItem{}, closeErr
	}

	item := FileItem{
		ID:        id,
		Name:      name,
		Size:      size,
		CreatedAt: time.Now(),
		StoredAs:  storedAs,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	files := append([]FileItem{item}, s.files...)
	if err := writeJSONAtomic(s.filesPath, files); err != nil {
		_ = os.Remove(dstPath)
		return FileItem{}, err
	}
	s.files = files
	s.advanceItemsETag()
	return item, nil
}

func (s *Store) FilePath(id string) (FileItem, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.files {
		if item.ID == id {
			return item, filepath.Join(s.uploadDir, item.StoredAs), true
		}
	}
	return FileItem{}, "", false
}

func (s *Store) DeleteFile(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.files {
		if item.ID == id {
			files := withoutItem(s.files, i)
			if err := writeJSONAtomic(s.filesPath, files); err != nil {
				return false, err
			}
			_ = os.Remove(filepath.Join(s.uploadDir, item.StoredAs))
			s.files = files
			s.advanceItemsETag()
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) AddMessage(text string) (MessageItem, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return MessageItem{}, errors.New("message is empty")
	}
	if len(text) > 65536 {
		return MessageItem{}, errors.New("message is too long")
	}
	id, err := newID()
	if err != nil {
		return MessageItem{}, err
	}
	item := MessageItem{ID: id, Text: text, CreatedAt: time.Now()}

	s.mu.Lock()
	defer s.mu.Unlock()
	messages := append([]MessageItem{item}, s.messages...)
	if err := writeJSONAtomic(s.messagesPath, messages); err != nil {
		return MessageItem{}, err
	}
	s.messages = messages
	s.advanceItemsETag()
	return item, nil
}

func (s *Store) DeleteMessage(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.messages {
		if item.ID == id {
			messages := withoutItem(s.messages, i)
			if err := writeJSONAtomic(s.messagesPath, messages); err != nil {
				return false, err
			}
			s.messages = messages
			s.advanceItemsETag()
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) CleanupExpired() error {
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionHours) * time.Hour)

	s.mu.Lock()
	defer s.mu.Unlock()

	isOldFile := func(item FileItem) bool { return item.CreatedAt.Before(cutoff) }
	if files := withoutExpired(s.files, isOldFile); len(files) != len(s.files) {
		if err := writeJSONAtomic(s.filesPath, files); err != nil {
			return err
		}
		for _, item := range s.files {
			if isOldFile(item) {
				_ = os.Remove(filepath.Join(s.uploadDir, item.StoredAs))
			}
		}
		s.files = files
		s.advanceItemsETag()
	}

	isOldMessage := func(item MessageItem) bool { return item.CreatedAt.Before(cutoff) }
	if messages := withoutExpired(s.messages, isOldMessage); len(messages) != len(s.messages) {
		if err := writeJSONAtomic(s.messagesPath, messages); err != nil {
			return err
		}
		s.messages = messages
		s.advanceItemsETag()
	}
	return nil
}

func withoutItem[T any](items []T, index int) []T {
	remaining := make([]T, len(items)-1)
	copy(remaining, items[:index])
	copy(remaining[index:], items[index+1:])
	return remaining
}

func withoutExpired[T any](items []T, expired func(T) bool) []T {
	count := 0
	for _, item := range items {
		if !expired(item) {
			count++
		}
	}
	if count == len(items) {
		return items
	}
	remaining := make([]T, 0, count)
	for _, item := range items {
		if !expired(item) {
			remaining = append(remaining, item)
		}
	}
	return remaining
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`)

func cleanFileName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	name = unsafeName.ReplaceAllString(name, "_")
	name = strings.Trim(name, ". ")
	if name == "" {
		return "upload.bin"
	}
	if len(name) > 160 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) > 20 {
			ext = ""
		}
		limit := 160 - len(ext)
		if limit < 1 {
			limit = 1
		}
		name = base[:min(len(base), limit)] + ext
	}
	return name
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
