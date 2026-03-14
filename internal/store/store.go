package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Ender-events/koncord/internal/domain"
)

// State is the persistent data model serialised to JSON.
type State struct {
	// Users maps platform user ID → role.
	Users map[string]domain.Role `json:"users"`
	// Bindings maps channel ID → container name.
	Bindings map[string]string `json:"bindings"`
}

// Store provides thread-safe, JSON-file-backed persistence.
type Store struct {
	mu       sync.RWMutex
	state    State
	filePath string
}

// New creates or loads a Store from the given file path.
func New(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		state: State{
			Users:    make(map[string]domain.Role),
			Bindings: make(map[string]string),
		},
	}

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read state file: %w", err)
		}
		if err := json.Unmarshal(data, &s.state); err != nil {
			return nil, fmt.Errorf("parse state file: %w", err)
		}
		// ensure maps are initialised
		if s.state.Users == nil {
			s.state.Users = make(map[string]domain.Role)
		}
		if s.state.Bindings == nil {
			s.state.Bindings = make(map[string]string)
		}
	}

	return s, nil
}

// ---------- User / Role helpers ----------

// GetRole returns the role for a given user ID (RoleNone if unknown).
func (s *Store) GetRole(userID string) domain.Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Users[userID]
}

// SetRole sets the role for a user and persists the state.
func (s *Store) SetRole(userID string, role domain.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Users[userID] = role
	return s.save()
}

// HasAnyUser returns true if at least one user is registered.
func (s *Store) HasAnyUser() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.Users) > 0
}

// ---------- Channel ↔ Container bindings ----------

// Bind associates a channel with a container name.
func (s *Store) Bind(channelID, containerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bindings[channelID] = containerName
	return s.save()
}

// GetBinding returns the container name bound to a channel (empty if unbound).
func (s *Store) GetBinding(channelID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bindings[channelID]
}

// ---------- internal ----------

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(s.filePath, data, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}
