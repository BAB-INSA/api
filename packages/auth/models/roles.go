package models

// Constantes pour les rôles disponibles
const (
	RoleUser       = "user"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "superAdmin"
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
		RoleSuperAdmin,
	}
}

// IsValidRole vérifie si un rôle est valide
func IsValidRole(role string) bool {
	validRoles := GetAllRoles()
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}
