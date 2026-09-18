package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store interface {
	Get(key string) (float64, bool, error)
	Put(key string, probability float64) error
}

type FileStore struct {
	Dir string
	mu  sync.Mutex
}

type entry struct {
	Probability float64 `json:"probability"`
}

func Key(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (s *FileStore) Get(key string) (float64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read cache: %w", err)
	}
	var value entry
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, false, fmt.Errorf("decode cache: %w", err)
	}
	if value.Probability < 0 || value.Probability > 1 {
		return 0, false, fmt.Errorf("cache probability outside [0,1]")
	}
	return value.Probability, true, nil
}

func (s *FileStore) Put(key string, probability float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create cache: %w", err)
	}
	if _, err := os.Stat(s.path(key)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect cache entry: %w", err)
	}
	data, err := json.Marshal(entry{Probability: probability})
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.Dir, ".reaper-cache-*")
	if err != nil {
		return fmt.Errorf("create cache entry: %w", err)
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write cache entry: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close cache entry: %w", err)
	}
	if err := os.Rename(name, s.path(key)); err != nil {
		return fmt.Errorf("publish cache entry: %w", err)
	}
	return nil
}

func (s *FileStore) path(key string) string {
	return filepath.Join(s.Dir, key+".json")
}

type Disabled struct{}

func (Disabled) Get(string) (float64, bool, error) { return 0, false, nil }
func (Disabled) Put(string, float64) error         { return nil }

type Memory struct {
	mu     sync.Mutex
	values map[string]float64
}

func NewMemory() *Memory { return &Memory{values: map[string]float64{}} }
func (m *Memory) Get(key string) (float64, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	return value, ok, nil
}
func (m *Memory) Put(key string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}
