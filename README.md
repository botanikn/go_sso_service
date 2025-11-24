# Dependencies

1. go compiler
2. docker
3. task utility

# Steps to run service

## First way

in go_sso_service folder:

1. docker-compose up -d sso_postgres
2. go mod download
2. task migrationsUp
3. task ssoServiceUp

## Second way (full docker)

in go_sso_service folder:

1. docker-compose up -d