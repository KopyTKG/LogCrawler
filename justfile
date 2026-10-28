# Linux here has gtk3/webkit2gtk-4.1 but not webkitgtk-6.0, so Go builds on Linux need the gtk3 tag.
tags := if os() == "linux" { "-tags gtk3" } else { "" }
bindflags := if os() == "linux" { "-f '-tags gtk3'" } else { "" }

default:
    @just --list

# install the wails3 CLI and the frontend dependencies
setup:
    go install {{tags}} github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
    yarn --cwd frontend install

# regenerate frontend/bindings after changing logservice.go
bindings:
    wails3 generate bindings {{bindflags}} -ts -i

# build the frontend into frontend/dist (embedded by the Go binary)
frontend: bindings
    yarn --cwd frontend build

# the version shown in the title bar: the git tag, else the short commit, else "dev"
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

# build bin/logcrawler
build: frontend
    go build {{tags}} -ldflags "-X main.version={{version}}" -o bin/logcrawler .

# build and run, optionally opening a log: just run testdata/raw/x.log
run *args: build
    ./bin/logcrawler {{args}}

# Go tests, the Svelte type check, and the frontend unit tests
test: bindings
    go test ./internal/...
    yarn --cwd frontend run check
    yarn --cwd frontend run test

# regenerate testdata/fixtures from testdata/raw through the redaction tool; READ THEM before committing
fixtures:
    go run ./cmd/redact -o testdata/fixtures testdata/raw/*.log

# eslint and a prettier check
lint:
    yarn --cwd frontend run lint
    yarn --cwd frontend run format:check
    gofmt -l internal cmd *.go

# format the frontend
fmt:
    yarn --cwd frontend run format
    gofmt -w internal cmd *.go

vet:
    go vet {{tags}} ./...
