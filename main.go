package main

import (
	"os"

	"github.com/ejoja/shortcuts/internal/app"
)

func main() {
	os.Exit(app.Main(os.Args, app.DefaultDeps()))
}
