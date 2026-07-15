---
name: jms-ssh
description: "Operate servers through the xops-infra/jms bastion using its non-interactive OpenSSH commands and the user's local public-key identity. Use when an AI needs to discover JMS-authorized targets or SSH users, run commands, stream stdin/stdout/stderr, inspect exit status, or transfer a small file through JMS without driving the TUI or using HTTP APIs."
---

# JMS SSH

Use the system `ssh` and `scp` clients. Let OpenSSH use the user's local key or `ssh-agent`; never read, copy, print, or upload a private key.

## Establish the endpoint

Use the JMS SSH alias or host, port, and login user supplied by the user or local SSH configuration. Do not guess an identity. Verify an alias without connecting:

```bash
ssh -G "$JMS_ALIAS" | awk '/^(hostname|user|port|identityfile) /'
```

Add `-o BatchMode=yes` for automation so missing public-key authentication fails instead of prompting for a password. The examples below use `$JMS_ALIAS`; replace it with ordinary OpenSSH connection arguments when no alias exists.

## Discover before executing

Narrow target discovery to avoid dumping a large inventory:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  targets --query "$TARGET_QUERY" --format json
```

Select an exact server ID whenever possible. Then discover its available upstream identities:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  users --target "$TARGET_ID" --format json
```

Never infer an SSH user or key when multiple choices remain. Use `--key KEY_NAME` if one username maps to multiple keys.

## Run commands

Run a normal command without a PTY:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" -- COMMAND ARG...
```

Set a timeout when appropriate; accepted values are 1 second through 30 minutes and the default is 60 seconds:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" --timeout 2m -- uname -a
```

Treat the local `ssh` exit status as authoritative. JMS returns the target command's exit status. Its own failures use:

- `2`: invalid arguments
- `64`: target missing or ambiguous
- `65`: public-key authentication or policy denied
- `66`: SSH user missing or ambiguous
- `67`: upstream connection/execution failure
- `68`: timeout

Keep stdout and stderr separate when structured handling matters:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" -- COMMAND \
  >stdout.log 2>stderr.log
status=$?
```

Pipe stdin directly for small textual input:

```bash
printf '%s' "$INPUT" | ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" -- wc -c
```

## Handle shell syntax safely

Use `--shell` for simple pipelines, redirections, or command lists. It consumes the remainder of the remote command:

```bash
ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" \
  --shell 'df -h | grep /data'
```

OpenSSH concatenates remote-command arguments, so nested quoting can become ambiguous. For scripts containing quotes, newlines, variables, or user-controlled values, encode the exact script and use `--shell-base64`:

```bash
script_b64=$(printf '%s' "$SCRIPT" | base64 | tr -d '\n')
ssh -o BatchMode=yes "$JMS_ALIAS" \
  run --target "$TARGET_ID" --user "$SSH_USER" \
  --shell-base64 "$script_b64"
```

Prefer direct argument mode for ordinary commands. Do not interpolate untrusted text into `--shell`.

## Transfer small files

JMS uses a custom SCP destination containing both the JMS login and target SSH identity:

```bash
# Upload
scp -P "$JMS_PORT" ./local-file \
  "$JMS_USER@$JMS_HOST:$SSH_USER@$TARGET_HOST:/remote/path"

# Download
scp -P "$JMS_PORT" \
  "$JMS_USER@$JMS_HOST:$SSH_USER@$TARGET_HOST:/remote/path" ./local-file
```

Verify important payloads with `sha256sum` through `run`. Compress directories first; JMS SCP does not support recursive directory transfer. For large binaries, prefer an approved internal artifact relay rather than sending bytes through model context or a slow public paste service.

## Guardrails

- Use `targets` and existing JMS policy as the source of authorization.
- Start with read-only inspection unless the user explicitly requested a mutation.
- Use `sudo -n`; never automate password prompts.
- Do not allocate a PTY or drive the two-level TUI with `expect` when these commands are available.
- Do not expose inventory, managed key material, command output, or internal hostnames beyond what the task requires.
- Preserve command exit status and report partial failures accurately.
