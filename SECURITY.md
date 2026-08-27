# Security policy

## Supported versions

| Version | Security fixes |
| --- | --- |
| 0.3.x | Supported |
| Earlier development builds | Not supported |

Use the newest published release before reporting a problem that may already be fixed.

## Report a vulnerability privately

Do not open a public issue with exploit details, tokens, customer data, a real Brief, or local paths.

1. Open a [private vulnerability report](https://github.com/Useless007/content-blueprint/security/advisories/new).
2. If private reporting is unavailable, open a public issue that asks the maintainer for a private contact channel. Do not include the vulnerability details in that issue.
3. Include the affected version, Windows/browser versions, a minimal synthetic reproduction, expected impact, and any workaround you already tested.

This is a volunteer-maintained project with no response-time SLA. The maintainer will confirm the report when available, reproduce it, agree on a disclosure plan, and credit the reporter if requested and appropriate.

## Useful security reports

Reports are especially useful when they concern:

- command or prompt injection that crosses the documented CLI boundary
- bypasses of `BriefRevision`, schema, size, or semantic validation
- unauthorized file read/write or environment leakage
- Native Messaging origin or message-validation bypasses
- extension behavior that reads Facebook data or triggers an external action without user intent
- updater acceptance of a wrong repository, asset, version, path, protocol, or digest
- installer behavior outside its per-user paths or documented registry keys
- a secret committed to repository history or release assets

Model quality, marketing performance, provider outages, and disagreement with generated wording are normally product issues rather than vulnerabilities. A model output that causes code execution, data disclosure, or a boundary bypass is a security report.

## Release integrity

Release assets include `SHA256SUMS.txt`, and the desktop updater verifies the exact installer digest before it can be launched. The checksum and installer currently come from the same GitHub Release, so this check detects file mismatch but does not establish publisher identity independently.

Version `v0.3.0` is not signed with Windows Authenticode. Verify the release URL and SHA-256 before running it. Code signing and a separate update trust anchor remain planned work.

## Secrets

Never put Claude, Codex, Gemini, GitHub, Meta, or Google credentials in source, issues, screenshots, test fixtures, shell transcripts, or logs. Revoke and rotate a credential immediately if it has been exposed; deleting the visible text does not remove it from Git history or notification copies.
