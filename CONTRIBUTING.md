# Contributing to Content Blueprint

Pull requests are welcome when they keep the user in control of publishing, messaging, audience creation, and update installation.

## Before opening code

- Search existing issues and describe the user problem first.
- Read [APP_SPEC.md](APP_SPEC.md), [CONTEXT.md](CONTEXT.md), and [docs/architecture.md](docs/architecture.md) for the current contracts and terms.
- Use synthetic Briefs and customer records in tests, screenshots, issues, and commits.
- For security problems, stop and follow [SECURITY.md](SECURITY.md).

Features that scrape profiles, followers, reactions, comments, inboxes, or customer lists from another page are outside project scope. So are automatic Post/Send actions, silent audience uploads, hidden update installs, and AI approval of business outcomes.

## Development setup

Requirements:

- Windows 10/11
- Go 1.26.7 or newer
- Node.js 22.12 or newer
- Wails CLI v2.15
- Chrome for UI smoke; Chrome and Brave for extension E2E
- NSIS for installer changes

```powershell
git clone https://github.com/Useless007/content-blueprint.git
cd content-blueprint
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0

cd frontend
npm ci
cd ..\facebook-extension
npm ci
cd ..

wails doctor
wails dev
```

Do not use a real `CONTENT_BLUEPRINT_DATA_DIR` during tests. The committed unit and browser fixtures are synthetic and do not require a live Facebook login or AI provider call.

## Required checks

Run the focused tests for your change, then:

```powershell
gofmt -w .\path\to\changed.go
cd frontend
npm ci
npm run build

cd ..
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

cd facebook-extension
npm ci
npm test
node --check service-worker.js
node --check sidepanel.js
node --check content-script.js
node --check src\core.js
node --check src\growth.js
```

Build `frontend/dist` before `go test`; the Wails entry point embeds that directory at compile time.

If you changed the extension, companion, Native Messaging, or browser handoff, run the [real-browser E2E instructions](facebook-extension/tests/e2e/README.md). The registration script writes only the documented current-user Native Messaging keys; remove the registration after local testing if you no longer need it.

If you changed layout or onboarding, build the frontend, start its preview server, then run `npm run test:wails-ui` from `facebook-extension`. Check desktop, 375 px mobile, short landscape, keyboard focus, and reduced motion.

If you changed packaging, install NSIS and run:

```powershell
.\build\windows\build-release.ps1
```

Do not publish or run the generated installer as part of a pull request test.

## Pull request shape

Keep each pull request focused. Explain:

- the user problem and the behavior after the change
- contracts or trust boundaries affected
- tests run and their results
- manual checks still needed
- screenshots for visible UI changes

Update README/docs and tests in the same pull request when behavior changes. Do not commit `node_modules`, build outputs, provider sessions, `.env` files, browser profiles, real customer data, or generated local state.

GitHub Actions must use official actions pinned to a reviewed full commit SHA. A dependency-pin update should name the corresponding action release tag in the comment.

## Commit and license terms

Use short, descriptive commits. By submitting a contribution, you agree that it is licensed under the repository's [MIT License](LICENSE) and that you have the right to submit it.
