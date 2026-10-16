# JMS access health and agent security

## Goal

Give people and internal agents only the access their work needs, make sensitive actions traceable, and alert an operator when access expands or auditing stops working.

## Current result

The review on 2026-10-14 verified production SSH and cluster connectivity and reviewed the access inventory. After a private database backup, a committed cleanup removed predictable email-derived local passwords and disabled expired grants and orphaned keys. Unnecessary debug-network exposure was narrowed. Administrator membership was preserved.

Application fixes pass local checks: authenticated administration, disabled-account enforcement, credential redaction, scoped file transfer, literal SCP paths and rejection when access policies cannot load. Directory sync no longer creates predictable local passwords. Application rollout must verify these boundaries on the deployed version; deployment validation results belong in the private operations report.

See [evidence, implementation and validation](2026-10-14-agent-security-design-detail.md).

## First round

1. Completed: verified connectivity and inventory, backed up the database, and cleaned confirmed obsolete or predictable access.
2. Completed locally: closed confirmed permission bypasses and checked ordinary password, LDAP and public-key access alongside rejection cases.
3. Rollout: verify allowed work, denied access, SSH authentication and audit behavior on the deployed version.
4. Follow up: review ownership of broad grants, shared credentials and automatic Team/Owner access; narrow justified grants and give temporary access an expiry.

Missing activity alone does not establish that an account is obsolete. A grant needs an owner and a current purpose; unclear ownership stays on the findings list until resolved.

## Next phases

**Protection:** give each agent a separate identity and credential with an owner, purpose and expiry. Separate discovery, command execution, file transfer and privileged target accounts. Keep human administration distinct from agent work. Privileged operations need a short-lived grant and a durable audit start record.

**Audit:** record who requested an action, which credential and target account were used, the permission decision, and the result. Include denied requests and user/group/key/policy changes. Redact credentials and protect the records from deletion by the acting identity.

**Alerts:** begin with administrator grants, new credentials, repeated denials, unusual target fan-out, bulk downloads and audit failures. Route actionable alerts to an agreed operator; group repeated events and verify delivery. Thresholds should follow observed normal activity.

## Scope and rollout

This round closes confirmed bypasses and performs targeted production access cleanup. A separate agent permission model, comprehensive security-event storage and alert delivery remain follow-up work.

Preview explicit-deny impact before rollout, then check representative ordinary users and administrators. Applicable denials also constrain administrators and Team/Owner grants. Existing automation must send authenticated administrative requests; self-service approval submissions remain available to signed-in users. Use managed secret storage for credential configuration and record runtime and access checks privately before accepting rollout.
