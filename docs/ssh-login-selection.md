# Automatic SSH login selection

JMS can connect without requiring the operator to know an image's login username in advance. Choose **Automatic connection** for a managed key in the SSH menu or Web interface. CLI users can omit `--user`; `--key` remains necessary when credentials are ambiguous.

JMS first tries the last successful username for that asset and key, then the configured key username, followed by `root`, `ubuntu`, and `ec2-user`. Each distinct username is attempted once. There is no guessing across different keys, and no image metadata is fabricated when the inventory does not provide it.

Only a rejected target SSH authentication advances to another username. Proxy errors, network failures, and command failures stop the attempt. A successful connection is selected before any command or file operation executes.

Explicit usernames and the existing per-asset `server.user` override remain strict. Password credentials are never enumerated. Operators can disable automatic probing with `sshLogin.autoUserEnabled: false`.

Successful usernames are learned separately from administrator settings and shared key metadata. The memory is scoped to asset ID, endpoint and public-key fingerprint; a rejected learned username is discarded. A changed key or endpoint does not reuse the old hint.

The release adds the connection behavior, menu and Web choices, accurate selected-user command audit, and shared login memory. It does not change shared key usernames or automatically rewrite server overrides.

Implementation and rollout notes are in [the detailed document](ssh-login-selection-detail.md).
