# Shell Task API Auth Update

Release date: 2026-03-13

This document describes the new authentication flow and related changes for the shell task API.

## What's New

- AD login endpoint that returns a JWT token valid for 1 day.
- JWT-based admin authorization for `POST /api/v1/shell/task`.
- `submit_user` is recorded on shell task creation (derived from the token).
- Improved error reporting when a target server has no SSH login user configured.

## AD Login (JWT)

Endpoint:

```
POST /api/v1/login/ad
```

Request (form or JSON):

```
{
  "user": "zhoushoujian",
  "password": "********"
}
```

Response:

```
{
  "token": "<jwt>",
  "expires_at": 1773471497
}
```

Token TTL is 24 hours.

## JWT Secret Configuration

Add the secret in `config.yaml`:

```yaml
withAuth:
  jwtSecret: "change-me"
```

If `jwtSecret` is empty, login and token validation will fail.

## Admin Authorization Rule

Only the shell task creation API is protected:

```
POST /api/v1/shell/task
```

The token subject must be a user in the database whose `groups` contains `"admin"`.
LDAP is used for authentication only and does not set admin groups automatically.

Example call:

```bash
curl -X POST http://localhost:8013/api/v1/shell/task \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "job-001",
    "servers": { "ip_addr": ["10.150.112.22"] },
    "shell": "echo \"hello world\""
  }'
```

## submit_user

When a shell task is created, `submit_user` is set from the JWT username.
You can see it in `GET /api/v1/shell/task`.

## Execution Behavior Notes

- Shell tasks are executed by the scheduler process. Make sure `jms scheduler` is running.
- If a target server has no configured SSH login user, the task will fail and record an error like:

```
no ssh user configured for server <ip>
```

Configure SSH login for the server via password or keypair before running tasks.
