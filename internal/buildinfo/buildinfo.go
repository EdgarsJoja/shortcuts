// Package buildinfo exposes version metadata set at build time.
package buildinfo

// These values are overridden with -ldflags at release time.
var (
	Version = "dev"
	Commit  = ""
)
