# Dependencies

1. Go compiler (1.24+)
2. Docker
3. [Task](https://taskfile.dev) utility

# Configuration

Non-secret settings (IP addresses, ports, DB name, timeouts, etc.) live in `config/config.yaml`. Secrets (DB username and password) are passed via environment variables in `.env`, which override the values from `config/config.yaml`. `.env` is not committed, so create it in the `go_sso_service` folder:

```
SSO_DB_USER=sso_user
SSO_DB_PASSWORD=<password>
SSO_DB_NAME=sso_db
```

# Steps to run the service

## First way

In the `go_sso_service` folder:

1. `docker compose up -d sso_postgres`
2. `go mod download`
3. `task migrationsUp`
4. `task ssoServiceUp` (it also applies migrations, so step 3 is optional)

## Second way (full Docker)

In the `go_sso_service` folder:

1. `docker compose up -d --build`

# Tests

`task test`

# Apps

Each app gets its own signing secret (at least 32 bytes). Secrets are generated automatically by the SSO service when you create an app.
Tokens are created and validated by calling the SSO service, so apps do not need to know their secret.

# How to start using SSO

1. Create an app with the `CreateApp` endpoint, passing the app name and the admin's credentials. The response contains the `app_id` of the created app. The admin user is registered with these credentials and becomes the admin of this app. The app name and the admin's email must not be taken yet.
2. Register users with the `Register` endpoint. A user gets the `user` permission in an app the first time they log in to that app with the `Login` endpoint, which returns an access token.
3. Endpoints that require authentication take the access token in the `authorization` metadata: `Bearer <token>`.
4. If you are an admin of an app, you can change users' permissions in this app with the `UpdatePermissions` endpoint.
5. There are only 3 permissions: `user`, `admin` and `banned`. Banned users cannot log in. An admin can get the permission of any user in the app with the `GetPermissionsByUserId` endpoint. Any user can get their own permission and user ID with the `CheckPermissionsByJwt` endpoint by passing their access token.
