# Переменные
CONFIG_PATH := config/config.yaml
MIGRATIONS_PATH := ./migrations

# Цель: запуск миграций
migrations-up:
	go run ./cmd/migrator --migrationsPath=$(MIGRATIONS_PATH)

# Цель: запуск SSO сервиса
sso-service-up: migrations-up
	SSO_CONFIG_PATH=$(CONFIG_PATH) go run ./cmd/sso --config=$(CONFIG_PATH)

# Цель по умолчанию
.PHONY: sso-service-up
