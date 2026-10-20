# SSH login selection implementation

[User-facing design](ssh-login-selection.md).

## Selection boundary

`model.SSHUser.Auto` marks an automatic request. `io.SelectSSHUser` preserves explicit usernames and rejects ambiguous credentials. `io.AutomaticSSHUsers` supplies automatic identities for implicit callers and compatible existing Web URLs; it no longer adds entries to interactive menus. Identical managed key material is deduplicated without exposing it.

`core/sshd.ConnectSSHClient` performs authentication and returns the actual selected identity. Session creation, shell execution, SFTP and SCP remain outside its retry loop. Its compatibility wrapper `NewSSHClient` keeps existing explicit call sites working.

Automatic candidates are the learned username, configured key username, root, ubuntu and ec2-user, with empty and duplicate names removed. At most five distinct usernames can be tried for one selected credential. Password identities, explicit selections, per-asset overrides and disabled probing make a single attempt.

Only a typed target_connect error whose underlying SSH error reports authentication rejection permits fallback. A proxy error containing similar text is never treated as a target rejection. No command is run to test usernames, and a failed command cannot restart authentication or execute again.

## Shared state

The additive ssh_login_hint table has a composite primary key of asset ID and public-key fingerprint, with endpoint, successful username and update time. It contains no private-key material. API startup migrates this model alongside the existing application schema.

Lookups also match the endpoint. Failed learned identities are conditionally invalidated by username so a stale attempt cannot delete a newer successful hint. Successful explicit or automatic key authentication updates the hint. Missing or unavailable hint storage is logged but does not block a valid SSH connection.

The existing server.user override remains separate and is preserved by asset ID during cloud synchronization. Neither learning nor enumeration edits shared key records.

## Clients and audit

- CLI run accepts an omitted --user; explicit --user remains strict. The command audit records the actual selected username.
- Interactive SSH menus show only the concrete username/key choices. Selecting one is strict and does not probe other usernames.
- The Web launch page requests ordinary ssh-users options and opens an explicit username/key route. Existing auth=auto workspace URLs still request automatic options for compatibility; new launch pages do not create those URLs.
- The Web terminal returns the successful username in an identity message and displays it in the connection snapshot. Reconnect and file operations retain the original automatic selection and selected key.
- Scheduled tasks and liveness probes use automatic authentication for their implicit username selection. Explicit SCP user@host paths remain strict.

## Configuration

```yaml
sshLogin:
  autoUserEnabled: false
```

Omitting this setting enables bounded username probing for automatic requests. Disabling it uses only the configured username; explicit usernames and administrator overrides work in either mode.

## Validation and rollout

Loopback SSH tests cover fallback, learned-first reuse, stale-hint invalidation, explicit/override/disabled/password single attempts, network/proxy stop conditions, cache failure tolerance, asset isolation and one-time command execution. DB tests cover fingerprint and endpoint isolation and concurrent conditional invalidation. API, CLI and IO tests cover selection semantics; the frontend must pass its TypeScript build.

Deploy the compatible Web build first, then API (creating the additive hint table), then SSHD and scheduler. The hint table may remain unused during rollback. No shared key or per-asset data rewrite is required for deployment.
