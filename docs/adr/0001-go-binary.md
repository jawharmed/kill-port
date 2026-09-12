# Ship killport as a Go binary

We need one `killport` command on macOS, Linux, and Windows, installed via GitHub Releases, Homebrew, and Scoop. Go gives straightforward cross-compilation to those three targets from a single CI job; Rust would work but with heavier builds, and a script/runtime (bash, Node) would not satisfy Windows and the install channels we chose.
