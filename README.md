# schemagen

Generate TypeScript, Python, and Go types from JSON Schema (JSON and YAML). One binary, zero dependencies.

## Install

### Homebrew (macOS/Linux)

```bash
brew tap mirpo/homebrew-tools
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

# YAML schemas work too
schemagen generate schemas/*.yaml --out-ts ./types
```

## Commands

### Global Flags

These flags are available for all commands:

- `-v, --verbose` - Enable verbose/debug logging
- `--json` - Output logs in JSON format

### generate

Generate code from JSON Schema files. Input can be a `.json`, `.yaml`, or `.yml` file, a directory, or a glob pattern.

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

**Flags:**
- `--no-color` - Disable colored diff output

## Schema Support

### Types
- **Objects** → structs / interfaces / BaseModel classes
- **Enums** → string/int/mixed enums (string unions in TS, `Enum`/`IntEnum` in Python, typed constants in Go)
- **Primitives** → native types with format handling (uuid, email, date-time, etc.)
- **Arrays / Maps** → typed slices/arrays/dicts
- **allOf** → struct embedding (Go), class inheritance (Python), `.extend()` (TS/Zod)
- **anyOf / oneOf** → union types (`Type1 | Type2` in TS, `Union[Type1, Type2]` in Python, `any` in Go)
- **$ref** → cross-file references with automatic import generation

### Constraints

JSON Schema validation constraints are propagated to generated code:

| Constraint | TypeScript (Zod) | Python (Pydantic) | Go (validator tags) |
|---|---|---|---|
| `minLength` / `maxLength` | `.min()` / `.max()` | `min_length` / `max_length` | `min` / `max` |
| `pattern` | `.regex()` | `pattern` | — |
| `minimum` / `maximum` | `.gte()` / `.lte()` | `ge` / `le` | `gte` / `lte` |
| `exclusiveMinimum` / `exclusiveMaximum` | `.gt()` / `.lt()` | `gt` / `lt` | `gt` / `lt` |
| `multipleOf` | `.multipleOf()` | `multiple_of` | — |
| `minItems` / `maxItems` | `.min()` / `.max()` | `min_length` / `max_length` | `min` / `max` |

### Input Formats
- JSON Schema (`.json`)
- YAML Schema (`.yaml`, `.yml`)

## What it generates

### TypeScript
- Interfaces with JSDoc comments and `@format` annotations
- Union types for anyOf/oneOf, string literal unions for string enums
- Auto-generated barrel exports (`index.ts`)

### Python
- Pydantic v2 BaseModel classes with full constraint validation
- Format types (EmailStr, UUID, datetime)
- Enum classes (`str, Enum` / `IntEnum` / `Literal` for mixed)
- Optional: `model_config = ConfigDict(extra='allow')` for additionalProperties (use `--py-additional-properties`)
- Auto-generated barrel exports (`__init__.py`)

### Go
- Structs with JSON tags and go-playground/validator tags
- Embedded structs for allOf composition
- UUID and time.Time format types
- Configurable pointer usage and package naming

## Validation Libraries

### TypeScript with Zod

Generate Zod schemas alongside TypeScript interfaces:

```bash
# Interfaces + Zod schemas
schemagen generate schema.json --out-ts ./types --ts-zod

# Only Zod schemas (with z.infer<> types)
schemagen generate schema.json --out-ts ./types --ts-zod-only

# Additional options
--ts-zod-strict        # Add .strict() to object schemas
--ts-zod-coerce-dates  # Use z.coerce.date() for date-time
```

### Python with Pydantic v2

Python output uses Pydantic v2 BaseModel with built-in validation:

- Field constraints: `min_length`, `max_length`, `pattern`, `ge`, `le`, `gt`, `lt`, `multiple_of`
- Format types: `EmailStr`, `UUID`, `datetime`
- Enums with string/int values

## License

MIT
