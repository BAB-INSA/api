package models

// Constantes pour les rôles disponibles
const (
	RoleUser      = "user"
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleEditor    = "editor"
)

// GetDefaultRoles retourne les rôles par défaut pour un nouvel utilisateur
func GetDefaultRoles() Roles {
	return Roles{RoleUser}
}

// GetAllRoles retourne tous les rôles disponibles
func GetAllRoles() []string {
	return []string{
		RoleUser,
		RoleAdmin,
		RoleModerator,
		RoleEditor,
	}
}
