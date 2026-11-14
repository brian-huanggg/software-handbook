package main

type Action string

const (
	ActionView   Action = "view"
	ActionDelete Action = "delete"
)

// Allowed is the only place that decides who may do what to a link.
// It is deny-by-default: anything not listed below is refused.
func Allowed(u User, a Action, l Link) bool {
	switch a {
	case ActionView:
		return l.OwnerID == u.ID
	case ActionDelete:
		return l.OwnerID == u.ID || u.Role == RoleAdmin
	default:
		return false
	}
}
