package migrations

func GetAllMigrations() []MigrationDefinition {
	migrations := []MigrationDefinition{}

	// Add core migrations
	migrations = append(migrations, GetCoreMigrations()...)

	// Ajoutez vos propres migrations ici

	return migrations
}
