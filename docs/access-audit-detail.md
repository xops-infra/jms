# Access audit implementation

[Behavior and rollout](access-audit.md)

## Data and entry points

`record_ssh_login` adds `source`, `ssh_user`, `ssh_key`. Source values are `web`, `ssh_interactive`, `ssh_noninteractive`; SQL NULL/empty remains unknown. Values and the caller are supplied by the server-side entry point, not a request parameter. Records are created after the upstream handshake has resolved the target identity; they do not guarantee that a shell or command subsequently started.

`record_ssh_command` adds `source`, `status`. A new `run` record starts with `status=running` before upstream connection. `completed` means the attempt ended, including connection failure and nonzero command exits. Completion conditionally updates a running UUID; it cannot overwrite a completed record. An automatic target username remains blank in the start record until resolved; completion stores the selected identity. Existing NULL/empty status remains historical. Unfinished records display no exit code.

The execution helper preserves the target exit status, including when completion cannot be saved. Internal audit-start failure uses `69` with an explanatory stderr message. A target command may independently return the same numerical exit code; use stderr and the record to distinguish it. Failures to save login records keep their existing log-and-continue behavior. This change makes CLI command starts mandatory; it does not make every audit path fail closed.

## API and UI

- `GET /api/v1/audit/login`: existing filters plus `source`; `unknown` matches NULL/empty and an invalid source returns 400.
- `GET /api/v1/audit/command`: administrator only; exact `user` and `ip`, `duration` in hours (default 24, range 1–8760), `limit` (default 50, maximum 200), nonnegative `offset`.
- Both return paged objects with `items`, `total`, `limit`, `offset`, `has_more`; order is creation time then numeric ID descending. Soft-deleted rows remain excluded.
- Existing command model field names retain their JSON capitalization; new fields `source` and `status` use lowercase. This preserves the stored model's existing representation.
- The command tab uses bounded pages and cancels stale requests. The range change resets its page and filters. Command text uses React text rendering inside a collapsed detail, not HTML or terminal interpretation. It is not automatically redacted; administrators must protect this sensitive data.

## Migration and checks

Existing Scheduler AutoMigrate includes both models. Apply the additive migration before new writers, then test old data and new source filters. Large audit tables may require a maintenance window for schema/index changes. A mixed rollout cannot provide complete new source coverage until all writers are updated. Do not backfill historical sources or status as if observed.

Tests cover source filters and actor/target retention, query authorization and pagination bounds, command lifecycle including conditional completion, start-write failure blocking execution, and completion-write failure preserving the command result without replay. Use local SQLite fixtures; legacy integration tests that require an operator database are not a safe default validation suite.

Test deployment acceptance: connect once through each entry point using an authorized test identity; verify the stored caller and actual target username, login filters and command result. Deny anonymous/non-admin audit queries. Interrupt a CLI session and inspect its record. Block audit writes in the test environment and confirm that a new command is not executed. Restarting production to perform these checks is not authorized by this document.
