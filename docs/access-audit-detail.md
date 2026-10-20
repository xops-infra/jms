# Access audit implementation

[Behavior and rollout](access-audit.md)

## Data and entry points

`record_ssh_login` adds `source`, `ssh_user`, `ssh_key`. Source values are `web`, `ssh_interactive`, `ssh_noninteractive`; SQL NULL/empty remains unknown. Values and the caller are supplied by the server-side entry point, not a request parameter. Records are created after the upstream handshake has resolved the target identity; they do not guarantee that a shell or command subsequently started.

`record_ssh_command` adds `source`, `status`. A new `run` record starts with `status=running` before upstream connection. `completed` means the attempt ended, including connection failure and nonzero command exits. Completion conditionally updates a running UUID; it cannot overwrite a completed record. An automatic target username remains blank in the start record until resolved; completion stores the selected identity. Existing NULL/empty status remains historical. Unfinished records display no exit code.

The execution helper preserves the target exit status, including when completion cannot be saved. A completion warning identifies the UUID on stderr and tells the caller not to retry automatically. Database error details are logged internally, not included in that warning or audit-start error. Internal audit-start failure uses `69`. A target command may independently return the same numerical exit code; use stderr and the record to distinguish it. Failures to save login records keep their existing log-and-continue behavior. CLI command starts are mandatory; not every audit path fails closed.

CLI `run` policy refusal writes `status=denied`, caller/fingerprint, requested target identity, command and the JMS refusal code 65. This occurs before credentials are resolved or an upstream connection starts. Invalid arguments, unknown targets and unavailable authorization context are not included in this denial path. Denial-write failures log an error and remain refusals.

## API and UI

- `GET /api/v1/audit/login`: existing filters plus `source`; `unknown` matches NULL/empty and an invalid source returns 400.
- `GET /api/v1/audit/command`: administrator only; exact `user` and `ip`, optional `status` (`running`, `completed`, `denied`, `failed`), `duration` in hours (default 24, range 1–8760), `limit` (default 50, maximum 200), nonnegative `offset`. `failed` selects completed records with nonzero exit code or timeout; it does not include denials. Invalid status returns 400.
- Both return paged objects with `items`, `total`, `limit`, `offset`, `has_more`; order is creation time then numeric ID descending. Soft-deleted rows remain excluded.
- Existing command model field names retain their JSON capitalization; new fields `source` and `status` use lowercase. This preserves the stored model's existing representation.
- The command tab uses bounded pages and cancels stale requests. The range change resets its page and filters. Command text uses React text rendering inside a collapsed detail, not HTML or terminal interpretation. It is not automatically redacted; administrators must protect this sensitive data.

## Migration and checks

Existing Scheduler AutoMigrate includes both models. Apply the additive migration before new writers, then test old data and new source filters. Large audit tables may require a maintenance window for schema/index changes. A mixed rollout cannot provide complete new source coverage until all writers are updated. Do not backfill historical sources or status as if observed.

Tests cover source filters and actor/target retention, query authorization and pagination bounds, command lifecycle including conditional completion, start-write failure blocking execution, and completion-write failure preserving the command result without replay. Use local SQLite fixtures; legacy integration tests that require an operator database are not a safe default validation suite.

Test deployment acceptance: connect once through each entry point using an authorized test identity; verify the stored caller and actual target username, login filters and command result. Deny anonymous/non-admin audit queries. Interrupt a CLI session and inspect its record. Block audit writes in the test environment and confirm that a new command is not executed. Restarting production to perform these checks is not authorized by this document.

## Release traceability

CI runs the local-fixture audit regressions before image publication. Main-branch image builds receive `VERSION` (date plus short commit) and `VCS_REF` (full commit) arguments; the binary reports the version and the final image carries `org.opencontainers.image.revision`. The Dockerfile makes only the public frontend directory readable/traversable for Nginx workers. A successful CI run or a new `latest` tag is not proof that every runtime has rolled; verify actual pod image IDs and reported binary versions.
