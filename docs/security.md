# Security

[Deployment hardening](deployment.md#first-setup-and-tls) · [Integration security](integrations.md) · [Development](development.md)

## Current limitations

**First-admin takeover is a known, deferred exposure risk.** A fresh instance accepts administrator setup from its first visitor, without a bootstrap token or application network gate. Isolate the host, container network and proxy until setup is complete. A loopback host mapping alone does not isolate peer containers. Do not recommend unattended public first boot. Secure session cookies do not fix setup authorization; implementation hardening remains [highest-priority roadmap work](roadmap.md#highest-priority).

The current source includes explicit `MEDIAGRAP_SECURE_SESSION_COOKIE`/`--secure-session-cookie` for trusted HTTPS termination. The published 0.0.6 image predates this and the new source-distribution/license changes; do not assume it contains these improvements or has the same security posture. There is no published supported-version/security-backport matrix yet. Review the exact version/digest you deploy; this document is not a penetration-test certificate.

The application uses a single-administrator model, not multi-user roles/tenant isolation. Provider settings and proxy credentials are sensitive SQLite data and are not promised encrypted at rest. Browser API reads omit secret values; logs must exclude them. Webhook signing secrets have a separate encryption key file, while MCP API Tokens are hashed; neither scheme makes a stolen config volume harmless.

## Protect a deployment

- Use TLS for non-local access and restrict direct backend access. Do not trust arbitrary forwarded headers. Keep setup/login on an intended origin; browser mutations use Session and CSRF, with HttpOnly/SameSite session cookies.
- Start with read-only media mounts. Mount only intended paths and set the container-visible allowlist. Run non-root, grant least privilege, and verify NAS ACL/symlink behavior before writes.
- Protect config backups, private environment/Compose overrides and separately mounted Webhook key files. Never put them in Git, Docker context, source archives, logs or screenshots. Rotate exposed provider keys/tokens and revoke sessions as appropriate to the incident.
- Keep `/source` available to remote users of distributed/modified builds; it serves only the matching baked archive, not local state. This is a deliberate unauthenticated route. Health/readiness expose safe aggregate status too; scope proxy access to probes if needed without blocking the source offer.
- Use deployment-owned Webhook host/CIDR policy; provider proxy settings do not relax it. MCP requires source-scoped Bearer credentials, exact permitted browser origins when Origin is present, and browser approval for movie file plans. Do not treat metadata text or tool output as authorization.
- Monitor disk capacity, mounts, readiness and integration backlog. Do not scan an accidentally empty mountpoint. Back up before writing/upgrading; individual atomic file replacement is not global rollback.

Provider metadata/images are untrusted input, and their service/content terms are separate from the application license. Runtime CSS currently requests Google Fonts; browser network/privacy/offline behavior may involve that third party. No asset-provenance or provider-terms certification is claimed here.

## Security-critical implementation boundaries

Regression tests must cover path traversal, symlink/root escape and concurrent writes; SSRF/redirect/DNS rebinding; bounded XML/image/probe input; session/CSRF and setup; token revocation/source grants; job cancellation/recovery and partial publication; and secret redaction/source-context exclusions. Use synthetic files and mock providers, not real libraries or credentials. NAS system calls may outlive cancellation; do not promise a timed-out operation never wrote a file.

Filesystem/SQLite atomicity is per supported primitive/transaction, not one distributed transaction. Browser rename/TV paths and the newer automation movie recovery engine have different guarantees. An ambiguous recovery record requires review; never blindly replay or delete surviving sources. See [architecture](architecture.md#mutations-jobs-and-integration-consistency).

## Reporting a vulnerability

No public GitHub destination or verified private reporting address has been configured in this documentation. **Do not send exploit details, credentials, database dumps or private media information to a public issue or an invented email address.**

If the actual hosting repository offers an enabled private vulnerability-reporting feature, use that verified feature. Otherwise use an already verified private maintainer channel; if none is available, request a private reporting route with a non-sensitive message and withhold details until it is established. Do not assume a public issue is confidential. The owner still needs to establish and publish this channel before inviting detailed vulnerability reports.

A private report should include the affected version/commit or image digest, concise impact, deployment assumptions and a minimal localhost/synthetic reproduction. Redact secrets and personal paths. Do not test against other people's deployments or upload real media/config to demonstrate an issue. No response-time or coordinated-disclosure deadline is promised before maintainers have configured the reporting process.

For ordinary non-sensitive bugs, follow [contributing guidance](development.md#proposing-a-change). Owners should separately configure private reporting, secret scanning/push protection where available, and main/dev branch protection; repository text cannot enable account features.
