package main

import (
	"os"
	"runtime/debug"

	"github.com/pyjeebz/why/cmd"
)

// version is stamped by goreleaser (-X main.version) on release builds.
var version = "dev"

func main() {
	os.Exit(cmd.Execute(resolveVersion()))
}

// resolveVersion falls back to the module version so go-installed
// binaries report their tag despite never seeing goreleaser's ldflags.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}
