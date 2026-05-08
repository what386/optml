package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const DefaultMetadataPath = "/opt/optml/metadata.json"

var ErrNotFound = errors.New("metadata entry not found")

type OptEntry struct {
	Name        string    `json:"name"`
	RootDir     string    `json:"root_dir"`
	BinPaths    []string  `json:"bin_paths"`
	Managed     bool      `json:"managed"`
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Checksum    string    `json:"checksum,omitempty"`
}

type MetadataState struct {
	Entries map[string]OptEntry `json:"entries"`
}

type MetadataStore struct {
	path string
	mu   sync.Mutex
}

func NewMetadataStore(path string) *MetadataStore {
	if path == "" {
		path = DefaultMetadataPath
	}
	return &MetadataStore{path: path}
}

func (s *MetadataStore) Path() string { return s.path }

func (s *MetadataStore) Load() (MetadataState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *MetadataStore) Save(state MetadataState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(normalizeState(state))
}

func (s *MetadataStore) List() ([]OptEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUnlocked()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(state.Entries))
	for k := range state.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]OptEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, state.Entries[k])
	}
	return out, nil
}

func (s *MetadataStore) Get(key string) (OptEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadUnlocked()
	if err != nil {
		return OptEntry{}, err
	}
	entry, ok := state.Entries[key]
	if !ok {
		return OptEntry{}, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return entry, nil
}

func (s *MetadataStore) Upsert(key string, entry OptEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	state.Entries[key] = entry
	return s.saveUnlocked(state)
}

func (s *MetadataStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if _, ok := state.Entries[key]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	delete(state.Entries, key)
	return s.saveUnlocked(state)
}

func (s *MetadataStore) loadUnlocked() (MetadataState, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MetadataState{Entries: make(map[string]OptEntry)}, nil
		}
		return MetadataState{}, fmt.Errorf("read metadata: %w", err)
	}
	if len(b) == 0 {
		return MetadataState{Entries: make(map[string]OptEntry)}, nil
	}
	var state MetadataState
	if err := json.Unmarshal(b, &state); err != nil {
		return MetadataState{}, fmt.Errorf("decode metadata: %w", err)
	}
	return normalizeState(state), nil
}

func (s *MetadataStore) saveUnlocked(state MetadataState) error {
	state = normalizeState(state)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir metadata dir: %w", err)
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	b = append(b, '\n')
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return fmt.Errorf("write temp metadata: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace metadata: %w", err)
	}
	return nil
}

func normalizeState(state MetadataState) MetadataState {
	if state.Entries == nil {
		state.Entries = make(map[string]OptEntry)
	}
	return state
}
