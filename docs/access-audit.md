# Access audit

JMS attributes access to the current authenticated caller and distinguishes three entry points: Web terminal, interactive SSH, and non-interactive SSH (Agent/CLI). A script run by a person uses the same non-interactive category; no separate Agent account is required.

Administrators can filter login records by entry point, caller, target and time. Login records include the resolved target SSH username and credential name. The command tab shows non-interactive `run` attempts, caller public-key fingerprint, target identity, completion, exit code, duration and timeout. Command text is available to administrators and may contain sensitive arguments.

Before a CLI command starts, JMS must save a durable start record. If this fails, it returns status `69` and does not connect upstream or execute. A completion-write failure retains an unfinished record and logs its UUID; JMS preserves the command result and does not retry it. Unfinished means the result needs investigation, not that the process is definitely still running.

Historical login sources remain unknown. Historical command records remain visible without a new lifecycle status. Existing records are not reclassified from guesses about their caller.

This release covers terminal access and CLI command execution. Discovery commands, rejected requests, administrative changes, security alerts and recording storage changes are separate follow-up work. Login records confirm an upstream connection, not successful completion of a command or shell.

Run schema migration with the new Scheduler version before upgrading SSHD/API, then upgrade the Web UI. The migration adds columns to existing audit tables; it does not delete historical records. Verify it in a test environment first and retain a backup. An unmigrated audit store can block CLI execution. Production rollout requires explicit operator authorization.

See [implementation and verification](access-audit-detail.md).
