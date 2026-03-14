package domain

// Role represents a user's permission level.
type Role int

const (
	RoleNone       Role = iota
	RoleUser            // can use status, restart, logs, list
	RoleAdmin           // can register users/admins, bind containers
	RoleSuperAdmin      // first user, full control
)

// String returns a human-readable role name.
func (r Role) String() string {
	switch r {
	case RoleSuperAdmin:
		return "super-admin"
	case RoleAdmin:
		return "admin"
	case RoleUser:
		return "user"
	default:
		return "none"
	}
}

// ContainerInfo holds information about a managed container.
type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string // e.g. "running", "exited"
	State  string // e.g. "Up 2 hours"
}
