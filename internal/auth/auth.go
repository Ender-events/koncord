package auth

import (
	"github.com/Ender-events/koncord/internal/domain"
	"github.com/Ender-events/koncord/internal/store"
)

// Manager handles authorisation checks and user registration.
type Manager struct {
	store *store.Store
}

// NewManager creates a new auth Manager backed by the given store.
func NewManager(s *store.Store) *Manager {
	return &Manager{store: s}
}

// EnsureInitialised auto-promotes the very first user who interacts to
// super-admin. Returns the role assigned.
func (m *Manager) EnsureInitialised(userID string) domain.Role {
	if !m.store.HasAnyUser() {
		_ = m.store.SetRole(userID, domain.RoleSuperAdmin)
		return domain.RoleSuperAdmin
	}
	return m.store.GetRole(userID)
}

// Authorise returns true if the user has at least the required role.
func (m *Manager) Authorise(userID string, required domain.Role) bool {
	role := m.store.GetRole(userID)
	return role >= required
}

// GetRole returns the current role of a user.
func (m *Manager) GetRole(userID string) domain.Role {
	return m.store.GetRole(userID)
}

// Register sets a user's role. Returns an error from the store layer.
func (m *Manager) Register(userID string, role domain.Role) error {
	return m.store.SetRole(userID, role)
}
