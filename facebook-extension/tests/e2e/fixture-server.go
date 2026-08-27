package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
)

const fixtureHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Content Blueprint local Facebook editor fixture</title>
  </head>
  <body>
    <main>
      <h1>Local editor fixture</h1>
      <label for="plainEditor">Textarea editor</label>
      <textarea id="plainEditor" aria-label="Textarea editor"></textarea>

      <div
        id="richEditor"
        role="textbox"
        aria-label="Contenteditable editor"
        contenteditable="true"
      ></div>

      <button id="postButton" type="button">Post</button>
    </main>
    <script>
      window.fixtureState = { inputEvents: 0, beforeInputEvents: 0, postClicks: 0 };
      document.addEventListener("input", () => { window.fixtureState.inputEvents += 1; }, true);
      document.addEventListener("beforeinput", () => { window.fixtureState.beforeInputEvents += 1; }, true);
      document.querySelector("#postButton").addEventListener("click", () => {
        window.fixtureState.postClicks += 1;
      });
    </script>
  </body>
</html>`

func main() {
	tmpl := template.Must(template.New("fixture").Parse(fixtureHTML))
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		if request.URL.Path != "/fixture.html" {
			http.NotFound(writer, request)
			return
		}
		_ = tmpl.Execute(writer, nil)
	}))
	server.StartTLS()
	defer server.Close()

	// The parent process replaces the loopback host with e2e.facebook.com and
	// supplies a Chromium host-resolver rule. This lets the unchanged production
	// hostname guard and manifest match run against a server that never leaves
	// this machine.
	fmt.Println(server.URL)
	_ = os.Stdout.Sync()
	_, _ = io.Copy(io.Discard, os.Stdin)
}
