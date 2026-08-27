# Real-browser extension E2E

This suite launches the installed Google Chrome and Brave binaries with a new,
isolated temporary profile for each run. It never opens a real Facebook account
or page and never clicks a publish control.

It verifies:

- the unpacked Manifest V3 extension and service worker load;
- Native Messaging health, brief sync, and current-pack retrieval;
- textarea and contenteditable insertion as literal plain text; and
- the fixture's `Post` button receives zero clicks.

## Run

From the repository root:

```powershell
go build -o build/bin/content-blueprint-companion.exe ./cmd/content-blueprint-companion
./facebook-extension/native-host/install-native-host.ps1
cd facebook-extension
npm ci
npm run test:e2e
```

Node.js 22.12 or newer is required by the browser driver. The suite uses
Puppeteer's extension installation API because branded Chrome removed the
`--load-extension` switch in Chrome 137. See the
[Chrome Extensions announcement](https://developer.chrome.com/blog/extension-news-june-2025#removing-the---load-extension-flag)
and [Puppeteer extension guide](https://pptr.dev/guides/chrome-extensions#at-runtime).

At runtime the suite copies only extension runtime files into a directory under
the operating-system temp folder. It adds loopback permissions only to that
copy; `facebook-extension/manifest.json` is checked to ensure it has no
localhost grant. The HTTPS fixture binds to `127.0.0.1` and is addressed through
`e2e.facebook.com` using a browser-only host-resolver rule, so the unchanged
Facebook hostname guard is exercised without network access to Facebook.

The browser profiles, temporary extension copy, and companion state are removed
after each browser run. Native-host registration is intentionally persistent and
can be removed with `native-host/uninstall-native-host.ps1`.
