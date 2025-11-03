# BAB-INSA API

API REST en Go pour l'association de baby foot BAB-INSA, développée avec le framework Gin et l'authentification JWT.

## Fonctionnalités

- ✅ Framework web Gin
- ✅ Authentification JWT pour les membres de l'association
- ✅ Base de données PostgreSQL avec GORM
- ✅ Système de migrations (comme Doctrine)
- ✅ Documentation Swagger/OpenAPI interactive
- ✅ Module d'authentification pour les joueurs
- ✅ Gestion des rôles utilisateurs (JSONB)
- ✅ Gestion des joueurs et tournois de baby foot
- ✅ Suivi des parties et statistiques
- ✅ Structure de projet organisée
- ✅ Configuration prête pour le développement

## Installation

1. Clonez le projet
```bash
git clone <repo-url>
cd api
```

2. Copiez et configurez les variables d'environnement
```bash
cp .env.example .env
# Éditez .env avec vos paramètres de base de données PostgreSQL
```

3. Installez les dépendances
```bash
go mod tidy
```

4. Exécutez les migrations
```bash
make migrate
```

## Utilisation

### Démarrer le serveur
```bash
make run
# ou
go run main.go
```

### Documentation API
Une fois le serveur démarré, accédez à la documentation Swagger interactive :
**http://localhost:8080/swagger/index.html**

### Endpoints disponibles

#### Authentification
- `POST /auth/register` - Inscription d'un nouveau membre
- `POST /auth/login` - Connexion membre
- `POST /auth/refresh` - Renouveler le token d'accès
- `POST /auth/logout` - Déconnexion membre
- `POST /auth/logout-all` - Déconnexion de tous les appareils (protégé)
- `POST /auth/change-password` - Changer le mot de passe (protégé)
- `POST /auth/reset-password/send-link` - Envoyer un lien de réinitialisation
- `POST /auth/reset-password/confirm` - Confirmer la réinitialisation

#### Membres
- `GET /users/me` - Profil du membre (protégé)
- `PUT /users/{id}` - Modifier email et username (protégé)

#### Autres
- `GET /health` - Health check
- `GET /protected/test` - Route de test protégée

### Compiler l'application
```bash
go build
```

### Migrations et fixtures

#### Migrations
```bash
make migrate          # Exécuter les migrations
make rollback         # Annuler la dernière migration
make rollback STEPS=3 # Annuler 3 migrations
make migration-status # Voir le statut des migrations

# Ou avec les binaires compilés en production
go build -o migrate-binary cmd/migrate.go
./migrate-binary migrate
./migrate-binary status
./migrate-binary rollback
```

#### Fixtures (données de test)
```bash
# Avec go run
go run cmd/fixtures.go generate    # Générer des données de test
go run cmd/fixtures.go clear       # Vider toutes les données
go run cmd/fixtures.go regenerate  # Vider et regénérer

# Ou avec un binaire compilé en production
go build -o fixtures-binary cmd/fixtures.go
./fixtures-binary generate
./fixtures-binary clear
./fixtures-binary regenerate
```

#### Déploiement
```bash
# 1. Compiler les binaires
go build -o bab-insa-api main.go
go build -o migrate-binary cmd/migrate.go
go build -o fixtures-binary cmd/fixtures.go   # Disponible mais pas exécuté automatiquement

# 2. Exécuter les migrations
./migrate-binary migrate

# 3. Démarrer l'API
./bab-insa-api

# 4. Fixtures (manuel, si besoin)
./fixtures-binary generate    # À exécuter manuellement selon les besoins
```

### Documentation
```bash
make swagger          # Régénérer la documentation Swagger
```

### Tests
```bash
make test             # Lancer les tests
# ou
go test ./...
```

### Autres commandes
```bash
make build            # Compiler l'application
make clean            # Nettoyer les fichiers générés
go fmt ./...          # Formater le code
```

## Structure du projet

```
.
├── main.go              # Point d'entrée de l'application
├── packages/auth/       # Module d'authentification réutilisable
│   ├── auth.go          # Configuration du module
│   ├── handlers/        # Handlers d'authentification
│   ├── middleware/      # Middleware JWT et rôles
│   ├── models/          # Modèles User, Claims, Roles
│   └── utils/           # Utilitaires JWT et passwords
├── handlers/            # Handlers Swagger et API
├── config/              # Configuration base de données
├── migrations/          # Système de migrations
│   ├── migration.go     # Moteur de migration
│   ├── auth_migrations.go # Migrations auth
│   └── migrations.go    # Vos migrations personnalisées
├── cmd/                 # Commandes CLI
│   └── migrate.go       # CLI de migration
├── docs/                # Documentation Swagger générée
├── .env.example         # Variables d'environnement exemple
├── Makefile             # Commandes make
└── go.mod              # Dépendances Go
```

## Technologies utilisées

- **Go 1.22+** - Langage de programmation
- **Gin** - Framework web rapide et minimaliste
- **GORM** - ORM pour Go
- **PostgreSQL** - Base de données
- **JWT** - Authentification par tokens pour les membres
- **Swagger/OpenAPI** - Documentation API
- **Go Modules** - Gestion des dépendances

## Licence

Apache 2.0