# schemagen

Generate TypeScript, Python, and Go types from JSON Schema. One binary, zero dependencies.

## Install

### Homebrew (macOS/Linux)

```bash
brew tap mirpo/tools
brew install schemagen
```

### Go

```bash
go install github.com/mirpo/schemagen@latest
```

### Binary

Download from [GitHub Releases](https://github.com/mirpo/schemagen/releases).

## Usage

```bash
# Generate TypeScript
schemagen generate schemas/ --out-ts ./types

# Generate Python (Pydantic v2)
schemagen generate schemas/ --out-py ./models

# Generate Go
schemagen generate schemas/ --out-go ./models

# All three at once
schemagen generate schemas/ --out-ts ./types --out-py ./models --out-go ./models
```

## Commands

### Global Flags

These flags are available for all commands:

- `-v, --verbose` - Enable verbose/debug logging
- `--json` - Output logs in JSON format

### generate

Generate code from JSON Schema files.

```bash
schemagen generate <input> [flags]
```

**Flags:**

Output directories:
- `--out-ts <dir>` - TypeScript output directory
- `--out-py <dir>` - Python output directory
- `--out-go <dir>` - Go output directory

General:
- `--extract-inline` - Extract inline enums and nested objects to top-level types
- `--output-strategy <strategy>` - Output strategy: `bundle`, `multifile`, or `bundledeps` (default: `multifile`)
- `--disable-headers` - Remove generated file headers
- `--disable-timestamp` - Remove timestamp from headers

TypeScript-specific:
- `--ts-unknown-any` - Use `unknown` instead of `any` for untyped schemas
- `--ts-additional-properties` - Add index signatures for additionalProperties

Python-specific:
- `--py-snake-case-field` - Convert field names to snake_case with JSON alias
- `--py-additional-properties` - Add `model_config = ConfigDict(extra='allow')` for schemas with additionalProperties

Go-specific:
- `--go-package <name>` - Package name for generated files (default: `models`)
- `--go-pointers` - Use pointers for optional fields (default: `true`)
- `--go-omit-empty` - Add omitempty to optional JSON tags (default: `true`)
- `--go-module-path <path>` - Go module path for absolute imports (e.g., `github.com/org/project`)

### validate

Check if schemas are valid JSON Schema.

```bash
schemagen validate schemas/
```

**Flags:**
- `--format <format>` - Output format: `text` or `json` (default: `text`)

### verify

Check if generated code matches schemas. Returns exit code 2 if drift detected. Useful for CI.

```bash
schemagen verify schemas/ --out-ts ./types
```

**Flags:**
- `--quiet` - Suppress output (only exit codes)

### diff

Show what would change if you regenerated.

```bash
schemagen diff schemas/ --out-ts ./types
```

## What it generates

### TypeScript
- Interfaces with JSDoc comments
- `@format` annotations (uuid, email, etc.)
- Union types for enums
- Auto-generated barrel exports (`index.ts`)

### Python
- Pydantic v2 BaseModel classes
- Field constraints (min, max, pattern)
- Format types (EmailStr, UUID, datetime)
- Enum classes
- Optional: `model_config = ConfigDict(extra='allow')` for additionalProperties (use `--py-additional-properties`)
- Auto-generated barrel exports (`__init__.py`)

### Go
- Structs with JSON tags
- Go validator tags (min, max, email, uuid, etc.)
- Embedded structs for allOf composition
- UUID and time.Time format types
- Configurable pointer usage and package naming

## License

MIT
