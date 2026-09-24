# Dependencies

1. go compiler
2. docker
3. task utility

# Configuration

Non-secret settings live in `config/config.yaml`. Secrets are passed via env variables:

```sh
cp .env.example .env   # then set SSO_DB_PASSWORD
```

Any `db.*` value can be overridden with `SSO_DB_HOST`, `SSO_DB_PORT`, `SSO_DB_USER`,
`SSO_DB_PASSWORD`, `SSO_DB_NAME`, `SSO_DB_SSLMODE`. TLS is enabled by setting
`grpc.tls_cert_file` and `grpc.tls_key_file` (or `SSO_TLS_CERT_FILE` / `SSO_TLS_KEY_FILE`).

# Steps to run service

## First way

in go_sso_service folder:

1. docker compose up -d sso_postgres
2. go mod download
3. task migrationsUp
4. task ssoServiceUp

## Second way (full docker)

in go_sso_service folder:

1. docker compose up -d --build

# Tests

task test

# Apps

Each app gets its own signing secret (at least 32 bytes). Migration 4 replaces secrets
shorter than that with random values; read the new one from the `apps` table if needed.
Tokens are validated by calling the SSO service, so apps do not need their secret.
