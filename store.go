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
	mu           sync.Mutex
	filesPath    string
	messagesPath string
	uploadDir    string
	files        []FileItem
	messages     []MessageItem
}

func NewStore(cfg Config) (*Store, error) {
	s := &Store{
		cfg:          cfg,
		filesPath:    filepath.Join(cfg.DataDir, "metadata.json"),
		messagesPath: filepath.Join(cfg.DataDir, "messages.json"),
		uploadDir:    filepath.Join(cfg.DataDir, "files"),
	}
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

func (s *Store) Items() ItemsResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ItemsResponse{
		Files:    append([]FileItem(nil), s.files...),
		Messages: append([]MessageItem(nil), s.messages...),
	}
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
	s.files = append([]FileItem{item}, s.files...)
	if err := writeJSONAtomic(s.filesPath, s.files); err != nil {
		_ = os.Remove(dstPath)
		return FileItem{}, err
	}
	return item, nil
}

func (s *Store) FilePath(id string) (FileItem, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.files {
		if item.ID == id {
			return item, filepath.Join(s.uploadDir, item.StoredAs), true
		}
	}
	return FileItem{}, "", false
}

func (s *Store) DeleteFile(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.files {
		if item.ID == id {
			_ = os.Remove(filepath.Join(s.uploadDir, item.StoredAs))
			s.files = append(s.files[:i], s.files[i+1:]...)
			_ = writeJSONAtomic(s.filesPath, s.files)
			return true
		}
	}
	return false
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
	s.messages = append([]MessageItem{item}, s.messages...)
	if err := writeJSONAtomic(s.messagesPath, s.messages); err != nil {
		return MessageItem{}, err
	}
	return item, nil
}

func (s *Store) DeleteMessage(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.messages {
		if item.ID == id {
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			_ = writeJSONAtomic(s.messagesPath, s.messages)
			return true
		}
	}
	return false
}

func (s *Store) CleanupExpired() error {
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionHours) * time.Hour)

	s.mu.Lock()
	defer s.mu.Unlock()

	files := s.files[:0]
	for _, item := range s.files {
		if item.CreatedAt.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.uploadDir, item.StoredAs))
			continue
		}
		files = append(files, item)
	}
	s.files = files

	messages := s.messages[:0]
	for _, item := range s.messages {
		if item.CreatedAt.Before(cutoff) {
			continue
		}
		messages = append(messages, item)
	}
	s.messages = messages

	if err := writeJSONAtomic(s.filesPath, s.files); err != nil {
		return err
	}
	return writeJSONAtomic(s.messagesPath, s.messages)
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
