// Package front serves the player's page: one static HTML file that talks to
// the api. Prototype: no build step, no framework.
package front

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var static embed.FS

// Handler serves the page and its assets.
func Handler() http.Handler {
	sub, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
