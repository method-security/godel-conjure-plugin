# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is godel-conjure-plugin, a [Gödel](https://github.com/palantir/godel) plugin for [conjure-go](https://github.com/palantir/conjure-go/). It generates Go code from Conjure API specifications and integrates with the Gödel build system.

## Development Commands

```bash
# Build the plugin
./godelw build

# Run all tests
./godelw test

# Run checks (formatting, linting, imports)
./godelw check

# Generate embedded CLI bundle (when updating Conjure version)
./godelw generate

# Format code
./godelw format

# Create distribution
./godelw dist

# Clean build artifacts
./godelw clean
```

## Plugin Usage Commands

```bash
# Generate Go code from Conjure specifications
./godelw conjure

# Verify generated code is up-to-date (non-destructive)
./godelw conjure --verify

# Publish Conjure IR to repository
./godelw conjure-publish --group-id=com.example --url=https://artifactory.example.com --repository=releases --username=user --password=pass
```

## Architecture

The plugin has two main entry points:
- **`conjure`**: Generates Go code from Conjure IR
- **`conjure-publish`**: Publishes Conjure IR to artifact repositories

### Key Components

- **`cmd/`**: CLI command definitions using Cobra
- **`conjureplugin/`**: Core plugin logic and IR providers
- **`conjureplugin/config/`**: Configuration parsing with version migration support
- **`ir-gen-cli-bundler/`**: Embedded Conjure CLI for YAML→IR conversion

### IR Source Support

The plugin supports multiple IR sources automatically detected by:
- **YAML directories**: Local Conjure YAML specifications
- **IR files**: Pre-generated Conjure IR JSON files  
- **HTTP URLs**: Remote IR files (https://host.com/ir.json)
- **AWS CodeArtifact**: Conjure IR packages from AWS CodeArtifact repositories

### Configuration

Plugin configuration is in `conjure-plugin.yml`:

```yaml
version: 1
projects:
  project-name:
    output-dir: generated/
    ir-locator: path/to/specs  # Auto-detects YAML dir, IR file, or URL
    accept-funcs: true         # Generate functional visitor patterns
    server: false              # Generate server code
    cli: false                 # Generate CLI code
    publish: true              # Allow IR publishing

  # AWS CodeArtifact example
  codeartifact-project:
    output-dir: generated/
    ir-locator:
      type: codeartifact
      codeartifact:
        domain: my-domain
        domain-owner: "123456789012"
        repository: my-repo
        package-group: com.example
        package: my-conjure-ir
        version: 1.0.0
        region: us-west-2      # Optional
        profile: my-profile    # Optional
```

## Build System

- Uses **Go 1.24.0** with vendor mode (`GOFLAGS: "-mod=vendor"`)
- Follows Gödel plugin conventions
- Embeds Conjure CLI using `go:embed` for YAML processing
- Integration tests create temporary projects and test full plugin execution

## Testing

Run integration tests specifically with:
```bash
./godelw test --tags=integration
```

Tests cover multiple IR source types and both generation/verification modes.