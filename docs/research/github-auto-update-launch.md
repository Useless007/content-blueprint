# Research: GitHub launch, Windows releases, and safe updates

Checked: 2026-08-28

Scope: official GitHub, GitHub CLI, Go, and Wails sources only. The current
project uses Wails v2.15.0. This note distinguishes facts stated by those
sources from implementation recommendations for Content Blueprint.

## Decision

Use a public GitHub repository and GitHub Releases as the distribution channel.
For Wails v2, implement **automatic update checks, not silent installation**:

1. Check at most once every 24 hours, plus a user-triggered "Check now".
2. Show the version, release notes, publisher/repository, file name, size, and
   verification status.
3. Download only after the user clicks.
4. Verify the exact expected asset and its digest/signature.
5. Ask again before visibly launching the Windows installer.
6. Quit the app only after the installer process starts successfully.

This is the safest useful interpretation of "auto update" for the first public
release. It avoids an unexpected install/restart and leaves Windows consent,
SmartScreen, and installer UI visible.

## 1. Public repository and release assets

GitHub CLI can create a public repository from an existing local repository;
`--source` selects the local source, `--remote` adds the remote, and `--push`
pushes the committed refs. A suitable launch command is:

```powershell
gh auth status
gh repo create OWNER/content-blueprint --public --source=. --remote=origin --push `
  --description "Local-first Facebook sales and SEO copilot for Windows"
```

Confirm the authenticated account, the intended owner/name, existing remotes,
license, `.gitignore`, and complete Git history before running it. A public
repository exposes its code, files, and revision history to everyone. GitHub
recommends a README, license, security features, and a security policy for a
public project. Sources: [GitHub CLI: `gh repo create`](https://cli.github.com/manual/gh_repo_create),
[About repositories](https://docs.github.com/en/repositories/creating-and-managing-repositories/about-repositories),
and [repository best practices](https://docs.github.com/en/repositories/creating-and-managing-repositories/best-practices-for-repositories).

Create and push the intended tag before publishing. `--verify-tag` is important:
without it, `gh release create` may create a missing tag from the default branch.
The installer and `SHA256SUMS.txt` can be attached in the same command:

```powershell
git tag -a v0.3.0 -m "Content Blueprint v0.3.0"
git push origin v0.3.0
gh release create v0.3.0 `
  build/bin/content-blueprint-amd64-installer.exe `
  build/bin/SHA256SUMS.txt `
  --verify-tag --generate-notes --latest
```

For an existing release, `gh release upload TAG FILES...` adds assets. Avoid
`--clobber` in routine release automation: the CLI documents that it deletes an
existing same-name asset before uploading the replacement, so a failed upload
can leave no asset. Sources: [GitHub CLI: `gh release create`](https://cli.github.com/manual/gh_release_create)
and [`gh release upload`](https://cli.github.com/manual/gh_release_upload).

In GitHub Actions, `gh` is installed on GitHub-hosted runners, but each step
that uses it must receive `GH_TOKEN`. Use the job's short-lived, repository-
scoped `${{ github.token }}`, not a PAT. `actions/upload-artifact` stores a
workflow artifact; it is not the public GitHub Release asset consumed by the
desktop updater. Sources: [using GitHub CLI in workflows](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-github-cli),
[`GITHUB_TOKEN`](https://docs.github.com/en/actions/concepts/security/github_token),
and [workflow artifacts](https://docs.github.com/en/actions/concepts/workflows-and-actions/workflow-artifacts).

## 2. Secure Windows release workflow

Recommended shape:

- Trigger the release job only from a deliberately pushed `v*` tag or a manual
  dispatch with a validated tag. Do not give a pull-request job release rights.
- Give test/build jobs `contents: read`. Give only the publishing job
  `permissions: contents: write`; when any permissions are specified, omitted
  permissions become `none`.
- Run the existing Go, frontend, extension, browser, and packaging checks before
  `gh release create`. Fail the job on any failed check or missing expected file.
- Build the NSIS installer on `windows-latest`, calculate SHA-256 after the final
  binary is produced, and upload both files to a draft before publication.
- Use the built-in `gh` command for publishing instead of adding a third-party
  release action. Pin every `uses:` action to a reviewed full commit SHA; GitHub
  calls a full SHA the only immutable action reference.
- Never interpolate untrusted GitHub context directly into a shell command.
  Pass it through an environment variable, validate it against the expected
  tag/version grammar, and quote it.
- Protect `v*` tags with a ruleset. Enable immutable releases so a published
  release locks its tag and assets. Complete all uploads while it is a draft,
  then publish it.
- Keep signing credentials only in Actions secrets or the relevant protected
  environment. Do not print, persist, or upload the decoded certificate.

GitHub documents job-level permissions and the `contents: write` scope in
[workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions),
full-SHA pinning and least privilege in the [secure-use reference](https://docs.github.com/en/actions/reference/security/secure-use),
and shell injection risk in [script injections](https://docs.github.com/en/actions/concepts/security/script-injections).
Tag rules are covered by [ruleset rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets),
and the draft-to-published protection is described in [immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases).

Wails v2 officially supports generating an NSIS installer with `wails build
-nsis`. Its signing guide recommends a normal Windows code-signing certificate
and shows CI signing with `signtool`. Sign both the application executable and
the final installer before public distribution; GitHub release integrity or an
attestation does not replace Windows Authenticode trust. Sources: [Wails v2 NSIS
installer](https://wails.io/docs/guides/windows-installer/) and [Wails v2 code
signing](https://wails.io/docs/guides/signing).

Optional hardening: GitHub artifact attestations establish build provenance and
can be verified with GitHub CLI. If adopted, grant only the documented OIDC and
attestation permissions to that job. This complements, rather than replaces,
Authenticode. Source: [Using artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations).

## 3. Latest-release API contract

For a public repository, this endpoint works without authentication:

```text
GET https://api.github.com/repos/OWNER/REPO/releases/latest
Accept: application/vnd.github+json
User-Agent: Content-Blueprint/<installed-version>
X-GitHub-Api-Version: 2026-03-10
If-None-Match: <saved-etag>   # after the first successful response
```

GitHub defines it as the latest published full release: drafts and prereleases
are excluded. It is selected by `created_at`, which is the release commit date,
not the publication date. Therefore compare normalized `tag_name` values using
SemVer and reject equal or lower versions; do not infer update order from dates.

Fields required by the updater:

| JSON field | Use |
| --- | --- |
| `tag_name` | Parse and compare the version. |
| `name`, `body`, `html_url` | Display release identity, sanitized notes, and a browser fallback. |
| `draft`, `prerelease` | Defense-in-depth checks even though `/latest` excludes both. |
| `published_at` | Display only; do not use for version ordering. |
| `assets[].name` | Exact allowlist match for the Windows installer. |
| `assets[].state` | Require `uploaded`. |
| `assets[].size` | Show the user and enforce a reasonable maximum before download. |
| `assets[].digest` | Expected SHA-256 when present. |
| `assets[].browser_download_url` | HTTPS download URL returned by GitHub; do not construct it manually. |

The official response schema demonstrates `digest` as `sha256:<hex>`, but the
documentation does not clearly guarantee a non-empty digest for every historic
asset. Keep `SHA256SUMS.txt` as a release asset and fail closed when the selected
installer cannot be verified. A checksum delivered through the same compromised
release account is not an independent authenticity proof, so retain Authenticode
verification and consider an application-embedded signing public key for a later
update-manifest design. Sources: [Get the latest release](https://docs.github.com/en/rest/releases/releases#get-the-latest-release)
and [release asset API](https://docs.github.com/en/rest/releases/assets).

All REST requests need a valid `User-Agent`; GitHub recommends the JSON accept
header and an explicit API version. Pinning the API version makes the response
contract deliberate. Sources: [Getting started with the REST API](https://docs.github.com/en/rest/using-the-rest-api/getting-started-with-the-rest-api)
and [API versions](https://docs.github.com/en/rest/about-the-rest-api/api-versions).

## 4. Rate limits, caching, and failure behavior

Unauthenticated public REST requests are limited to 60 per hour per originating
IP. Several users behind one NAT can share that IP budget. Do not embed a PAT,
OAuth client secret, or release credential in a desktop binary merely to raise
the limit.

Store the response `ETag` and send it as `If-None-Match`; handle `304 Not
Modified` without parsing a body. However, GitHub only promises that a `304`
does not count against the primary limit when the conditional request is
correctly authorized. An unauthenticated desktop app must not assume its `304`
is free.

Recommended policy:

- no more than one automatic check per 24 hours per installation;
- coalesce concurrent checks and offer a separate manual check;
- persist `ETag`, last successful check, and next allowed check locally;
- honor `retry-after`, `x-ratelimit-remaining`, and `x-ratelimit-reset`;
- on `403`/`429`, stop until the indicated time, then use exponential backoff;
- on `404` or repeated `5xx`, show a non-blocking status and keep the installed
  application working;
- apply connection, header, body, and total-download timeouts; cap response and
  asset sizes.

Sources: [REST API rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
and [REST API best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api).

GitHub also supports a stable browser URL ending in
`/releases/latest/download/<asset-name>`. It is suitable for a user-facing
download fallback, but the application still needs trusted version metadata and
verification before executing a file. Source: [Linking to releases](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases).

## 5. Wails v2 implementation boundary

The current repository imports `github.com/wailsapp/wails/v2 v2.15.0`. The
official v2 runtime reference documents browser opening, dialogs/events, and
`runtime.Quit`; the Windows guide shows spawning a separate process with Go's
`exec.Command`; the installer guide covers creating NSIS packages. These v2
sources do not document an application self-updater. Wails v3 now documents an
updater, but that is a different major version and its API must not be assumed
to work in this v2 application. This is an inference from the official v2 API
surface, not a claim that no community updater exists.

Sources: [Wails v2 runtime](https://wails.io/docs/reference/runtime/intro/),
[opening a system browser](https://wails.io/docs/reference/runtime/browser/),
[Windows process spawning](https://wails.io/docs/guides/windows/), and [Wails v3
updater FAQ](https://v3.wails.io/faq/).

Recommended v2 sequence:

1. A small Go update service owns HTTP, caching, SemVer comparison, download,
   and verification. The React UI receives typed status only.
2. Validate the release host, exact asset name, file size, digest, and version;
   save to a newly created per-update temporary directory, never the install
   directory.
3. Render release notes as plain text or sanitized Markdown. Never inject API
   `body` as raw HTML.
4. On "Install now", show the final confirmation and start the NSIS installer as
   a **visible** child process. Do not pass a silent-install flag.
5. If process start succeeds, call `runtime.Quit`. If it fails, keep the app open
   and preserve a useful error. A separate installer is necessary because the
   running application should not overwrite itself.
6. On the next launch, report the installed version and clean only update temp
   directories owned by this application.

The exact in-place upgrade and rollback behavior depends on this project's
custom NSIS script and is not guaranteed by Wails. Test upgrade, cancel,
interrupted download, invalid digest, installer launch failure, downgrade,
files-in-use, and retained user data on a clean Windows VM before calling the
feature production-ready.

## 6. Release acceptance checklist

- Public-history and secret scan completed; no credentials, personal datasets,
  local paths, or generated private state are tracked.
- License, README screenshots, privacy/security boundaries, `SECURITY.md`, and
  contribution instructions are present.
- Tag points at the reviewed commit; tag/version/installer metadata agree.
- All tests pass on the release commit.
- Installer and executable are Authenticode-signed and timestamped, or the
  release clearly discloses that they are unsigned and may trigger SmartScreen.
- SHA-256 file is generated after signing and matches the uploaded installer.
- Release is assembled as a draft, assets verified, then published as stable and
  latest; immutable releases and tag protection are enabled.
- Fresh install and upgrade from the previous release pass on a clean Windows VM.
- Update check handles 200, 304, 403, 404, 429, timeout, malformed JSON, missing
  asset, oversized asset, bad SemVer, bad digest, and canceled user confirmation.
