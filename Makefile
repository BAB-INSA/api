# Makefile pour le projet bab-insa-api

.PHONY: help build run dev migrate rollback migration-status swagger test clean lint quality

# Variables
APP_NAME=bab-insa-api
BUILD_DIR=./build

help: ## Afficher l'aide
	@echo "Commandes disponibles:"
	@echo "  build            - Compiler l'application"
	@echo "  run              - Lancer l'application"
	@echo "  dev              - Lancer en mode développement (auto-rebuild)"
	@echo "  migrate          - Exécuter les migrations"
	@echo "  rollback [STEPS] - Annuler les migrations (défaut: 1)"
	@echo "  migration-status - Afficher le statut des migrations"
	@echo "  swagger          - Générer la documentation Swagger"
	@echo "  test             - Lancer les tests"
	@echo "  lint             - Lancer golangci-lint"
	@echo "  quality          - Lancer tous les outils de qualité"
	@echo "  clean            - Nettoyer les fichiers générés"

build: ## Compiler l'application
	@echo "Compilation de $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) .

run: ## Lancer l'application
	@echo "Démarrage de $(APP_NAME)..."
	go run main.go

dev: ## Lancer en mode développement avec auto-rebuild
	@echo "Démarrage en mode développement..."
	@which air >/dev/null 2>&1 || (echo "Air non trouvé. Installation..."; go install github.com/air-verse/air@latest)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		$(shell go env GOPATH)/bin/air; \
	fi

migrate: ## Exécuter les migrations
	@echo "Exécution des migrations..."
	go run cmd/migrate.go migrate

rollback: ## Annuler les migrations (usage: make rollback STEPS=2)
	@echo "Annulation des migrations..."
	@if [ -z "$(STEPS)" ]; then \
		go run cmd/migrate.go rollback; \
	else \
		go run cmd/migrate.go rollback $(STEPS); \
	fi

migration-status: ## Afficher le statut des migrations
	@echo "Statut des migrations:"
	go run cmd/migrate.go status

swagger: ## Générer la documentation Swagger
	@echo "Génération de la documentation Swagger..."
	@which swag >/dev/null 2>&1 || (echo "swag not found in PATH, trying common locations..."; /home/magicbart/gocode/bin/swag init && exit 0)
	swag init

test: ## Lancer les tests
	@echo "Exécution des tests..."
	go test ./...

clean: ## Nettoyer les fichiers générés
	@echo "Nettoyage..."
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)

# Commandes pour le développement
dev-setup: ## Configuration initiale pour le développement
	@echo "Configuration du projet pour le développement..."
	cp .env.example .env
	@echo "⚠️  N'oubliez pas de configurer le fichier .env avec vos paramètres de base de données"

tidy: ## Nettoyer les dépendances Go
	go mod tidy

# Outils de qualité de code
lint: ## Lancer golangci-lint
	@echo "Analyse du code avec golangci-lint..."
	@which golangci-lint >/dev/null 2>&1 || $(shell go env GOPATH)/bin/golangci-lint run; \
	if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else $(shell go env GOPATH)/bin/golangci-lint run; fi

quality: ## Lancer tous les outils de qualité de code
	@echo "=== Formatage du code ==="
	gofmt -s -w .
	@echo "=== Vérification Go ==="
	go vet ./...
	@echo "=== Analyse statique ==="
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; else $(shell go env GOPATH)/bin/staticcheck ./...; fi
	@echo "=== Scan de sécurité ==="
	@if command -v gosec >/dev/null 2>&1; then gosec ./... || true; else $(shell go env GOPATH)/bin/gosec ./... || true; fi
	@echo "=== Linter ==="
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run || true; else $(shell go env GOPATH)/bin/golangci-lint run || true; fi
	@echo "✅ Tous les outils de qualité ont été exécutés"