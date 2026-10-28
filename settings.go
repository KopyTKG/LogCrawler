package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

const maxRecent = 8

// Settings are the few things worth remembering between runs. They live in the user's config
// directory and hold paths and sizes only, never log content.
type Settings struct {
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	LastDir string   `json:"lastDir"`
	View    string   `json:"view"`
	Recent  []string `json:"recent"`

	mu   sync.Mutex
	path string
}

func loadSettings() *Settings {
	s := &Settings{Width: 1200, Height: 780, View: "screens"}
	dir, err := os.UserConfigDir()
	if err != nil {
		return s
	}
	s.path = filepath.Join(dir, "LogCrawler", "settings.json")
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, s) // a broken file just means defaults
	}
	if s.Width < 640 || s.Height < 400 {
		s.Width, s.Height = 1200, 780
	}
	return s
}

func (s *Settings) save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		return
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil || os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	_ = os.WriteFile(s.path, b, 0o644)
}

// opened records a file as the most recent one.
func (s *Settings) opened(path string) {
	s.mu.Lock()
	s.LastDir = filepath.Dir(path)
	s.Recent = slices.DeleteFunc(s.Recent, func(p string) bool { return p == path })
	s.Recent = append([]string{path}, s.Recent...)
	if len(s.Recent) > maxRecent {
		s.Recent = s.Recent[:maxRecent]
	}
	s.mu.Unlock()
	s.save()
}
