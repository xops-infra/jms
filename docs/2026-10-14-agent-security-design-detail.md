# JMS access health and agent security — detail

The [main design](2026-10-14-agent-security-design.md) owns scope and rollout decisions. This document contains implementation evidence and technical follow-up. Raw production inventory, credentials and internal endpoints remain outside this open-source repository.

## Implementation and rollout status

The application changes below are implemented and verified with local fixtures. Production SSH and cluster connectivity and the access inventory have been verified. A cleanup transaction was committed after a private database backup: email-derived local passwords were removed, expired enabled grants were disabled, and orphaned keys were disabled. Administrator membership was preserved.

The live cleanup and local tests do not prove enforcement by a deployed build. Deployment validation results are recorded in the private operations report. Application rollout must verify administrative authentication, active-account SSH access, policy rejection and audit behavior against the running version.

## Baseline findings

The findings describe behavior identified before this round's fixes. Treatments marked as local are implemented and tested; live enforcement requires verification against the deployed build.

| Priority | Finding | Local evidence | Treatment |
| --- | --- | --- | --- |
| Critical | Anonymous user/group, approval decision and key/profile/proxy management | `core/api/rest.go`, corresponding handlers | Local: administrator authentication and authenticated approval submission |
| Critical | Stored user/proxy/key/cloud credentials in responses | Credential models and corresponding handlers | Local: omit secrets from responses while retaining request/runtime credential handling |
| Critical | Transfer checks ignore non-IP filters or authorize a different host | `io/policy.go: CheckPermission`, `io/sshd.go: parseScpPath` | Local: authorize the actual connection host and evaluate every asset filter |
| Critical | SCP remote paths become shell code | `core/sshd/scp.go: scpRemoteCommand` | Local: quote one literal path argument and terminate option parsing |
| High | Failed policy loads can leave implicit access available | `io/policy.go: GetUserPolicys` and its callers | Local: propagate load errors and reject access before connecting |
| High | Implicit grants override explicit denies | `io/policy.go: MatchPolicy / SystemPolicyCheck` | Local: applicable denies override admin, Team and Owner grants |
| High | Wildcard group values grant admin | `model/db.go: ArrayString.Contains`, exact role checks | Local: require literal admin membership; keep patterns for asset policies |
| High | Deleted accounts can remain authorized | API authorization, DB authentication, SSH authentication and key registration | Local: enforce active accounts across those paths |
| High | Directory sync creates an email-derived local password | `core/dingtalk/workflow.go: saveDingtalkUsers`, `core/db/user.go: Login` | Local: provision no default local password and reject unset passwords; live defaults removed |
| High | Credentials enter proxy errors and startup/debug logs | `core/sshd/sshd.go`, `core/api/key.go`, `app/state.go` | Local: remove credential dumps from proxy errors and proxy/key/database/DingTalk logs |
| High | Rejected operations are not consistently audited; successful execution can outlive audit failure | `cmd/sshd_noninteractive.go: runCommand`, `core/api/terminal.go` | Follow-up start/decision/completion events and audit-failure handling |
| High | No common administration event trail or access-misuse alert rules | Management handlers, `core/api/shell.go`, `core/liveness.go`, `core/shell.go` | Follow-up structured events and actionable alerts |
| High | Reversible local passwords and permissive host trust | `core/db/user.go`, `core/sshd/sshd.go` | Follow-up password migration and host-key verification |
| High | Web terminal accepts every origin and interpolates tmux session identifiers | `core/api/terminal.go`, `web/src/components/TerminalView.tsx` | Follow-up origin policy, literal session arguments and short-lived terminal tickets |

Potential historical disclosure requires a separate review of network reachability, API/log exposure and affected credentials. Hiding future response fields does not rotate previously exposed secrets.

## First patch boundaries

- Require the exact `admin` role for management APIs; reject absent/invalid credentials and deleted users. Wildcard syntax stays available for asset policies, not role membership.
- Bind approval applicant and decision attribution to the authenticated actor. Keep self-service submission authenticated.
- Reject disabled accounts in DB password login, SSH password/LDAP authentication, public-key authentication, key registration, API authorization and asset permission decisions. Existing active LDAP and key users do not need a local password.
- Directory sync creates account metadata without a local password and preserves existing credentials, groups and disabled status. Unset or empty stored passwords cannot authenticate locally.
- Evaluate active, nondeleted, unexpired policy denies before returning implicit access. A deny wins even if the same policy also contains an allow.
- Share transfer target parsing between authorization and connection, then evaluate ID, name, IP, environment, Team and custom tags together on the loaded asset.
- Propagate policy-load failures through transfer, server discovery, terminal, interactive menu and command paths. An unavailable policy store cannot silently become an empty policy list with implicit access.
- Construct upstream SCP commands with a quoted literal path and `--`. Shell operators, substitutions and leading option characters stay within the path argument. Shell expansion of variables, home-directory shortcuts and globs is no longer supported.
- Omit stored credentials from public responses, while preserving credential input and internal runtime loading. Proxy errors and selected debug/startup logs retain operational context without passwords, private keys or full connection credentials.
- Preserve current Team/Owner implicit grants when no deny applies. Separating execution and privileged account scope is a later product decision.

Debug profiling now binds to loopback, so ordinary network clients cannot reach it. Network-policy configuration alone is insufficient evidence of enforcement; validate traffic rejection on the selected cluster.

## Production health evidence and cleanup

Connectivity and inventory review are complete. The committed cleanup removed known email-derived local passwords and disabled expired enabled grants and orphaned keys after a private database backup. Unnecessary debug-network exposure was narrowed. The cleanup preserved administrator membership and did not classify an account as obsolete from missing activity alone. Raw evidence and recovery materials remain private.

Use local public-key authentication with `BatchMode=yes`. Discover the exact authorized target and available target account with `targets` and `users` before `run`. Follow [jms-ssh](../.agents/skills/jms-ssh/SKILL.md); a backup alias alone does not establish the production entry.

Continue reviewing credential ownership and usage, broad filters, Team/Owner matches, privileged target accounts, transfer reachability and API/database/debug exposure. Keep raw account and network details in an approved private location outside this open-source repository.

Preview explicit-deny impact before application rollout. Use managed secret storage for credential configuration and verify that the running application receives the approved configuration. Compare allowed and denied access for representative ordinary users, agents and administrators, and confirm deployed version, SSH authentication and audit behavior. Record results and grant removals with actor, rationale, affected scope and rollback information in the private operations report. Preserve recovery access and leave unresolved ownership on the findings list.

## Audit and alert follow-up

An event should identify the actor, agent owner where applicable, credential fingerprint, source, request/session ID, target and target account, action, permission decision, timing and result. Administrative events include a redacted change summary. A command's start record must exist before privileged execution; completion updates must distinguish failure, timeout, disconnect and unknown outcome.

Keep secrets, raw tokens and private keys out of events. Command arguments and terminal output can themselves contain secrets and need a documented redaction/access policy. Store audit records outside the acting identity's modification rights; choose retention and export requirements with operators.

Begin alert rules with privilege grants, new keys, repeated denies, unusual access breadth, bulk downloads and audit-pipeline failure. Determine thresholds from the live baseline; deduplicate bursts, track recovery, and expose delivery failures. No notification recipient or sending action is configured by this design.

These directions follow [OWASP Authorization guidance](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) on checking every request and default denial, and [OWASP Logging guidance](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) on recording authentication/authorization failures while excluding secrets.

## Validation boundaries

Local regression checks cover management authentication, actor attribution, exact admin roles, disabled accounts, active password/LDAP/key access, directory sync without default passwords, sanitized responses and proxy errors, literal SCP arguments, explicit deny precedence, actual transfer host/full asset scope and policy-load failure. Internal credential loading and existing account state remain covered. These checks do not prove production application enforcement.

Legacy external tests in `io/instance_test.go` and `core/db/service_test.go` initialize an externally configured application; some mutate its database or contact cloud providers. Run the new policy and DB tests against explicit source-file lists to avoid those initializers. The required `make web` exercises the local Compose build/start path only. Production requires its own version, denied-request, allowed-work and audit checks after deployment.
