# Installation

## Homebrew (macOS/Linux)

```bash
# Install the full suite
brew install e1sidy/tap/kite

# Or install individually
brew install e1sidy/tap/slate    # task layer only
brew install e1sidy/tap/roland   # orchestrator (installs slate as dependency)
```

## Go Install

```bash
go install github.com/e1sidy/slate/cmd/slate@latest
go install github.com/e1sidy/roland/cmd/roland@latest
```

Requires Go 1.25+.

## Build from Source

```bash
# Slate
git clone https://github.com/e1sidy/slate.git
cd slate && go build -o slate ./cmd/slate/

# Roland (requires Slate as sibling directory)
git clone https://github.com/e1sidy/roland.git
cd roland && go build -o roland ./cmd/roland/
```

## Verify Installation

```bash
slate version
roland version
```

## Post-Install Setup

```bash
# Initialize Slate
slate config init

# Initialize Roland
roland init

# Register a repository
roland repo add https://github.com/your-org/backend.git
```

See [Getting Started](getting-started.md) for a full walkthrough.
