#!/usr/bin/env bash
set -e

cd "$(dirname "$0")/.."

echo "=== Linting testdata/expected/ ==="
echo ""

# ============================================================================
# VALIDATE TypeScript
# ============================================================================
echo "=== Validating TypeScript (default) ==="
if npx --package typescript tsc --noEmit $(find testdata/expected/ts -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (default) validation PASSED"
else
    echo "❌ TypeScript (default) validation FAILED"
    exit 1
fi

echo ""
echo "=== Validating TypeScript (extracted) ==="
if npx --package typescript tsc --noEmit $(find testdata/expected/ts-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (extracted) validation PASSED"
else
    echo "❌ TypeScript (extracted) validation FAILED"
    exit 1
fi

# ============================================================================
# VALIDATE TypeScript with Zod (requires temporary zod installation)
# ============================================================================
echo ""
echo "=== Setting up Zod for TypeScript validation ==="

# Create a temp directory with zod installed for type checking
ZOD_TEMP_DIR=$(mktemp -d)
trap "rm -rf $ZOD_TEMP_DIR" EXIT

# Initialize minimal package.json and install zod
cd "$ZOD_TEMP_DIR"
npm init -y >/dev/null 2>&1
npm install --save-dev zod typescript >/dev/null 2>&1
cd - >/dev/null

echo ""
echo "=== Validating TypeScript (ts-zod-extracted) ==="
if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution bundler --rootDir "$ZOD_TEMP_DIR" --paths '{"zod": ["'"$ZOD_TEMP_DIR"'/node_modules/zod"]}' $(find testdata/expected/ts-zod-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (ts-zod-extracted) validation PASSED"
else
    # Fallback: try with ignoreDeprecations for older TS versions
    if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution node --baseUrl "$ZOD_TEMP_DIR/node_modules" --ignoreDeprecations 6.0 $(find testdata/expected/ts-zod-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
        echo "✅ TypeScript (ts-zod-extracted) validation PASSED"
    else
        echo "❌ TypeScript (ts-zod-extracted) validation FAILED"
        exit 1
    fi
fi

echo ""
echo "=== Validating TypeScript (ts-zod-only-extracted) ==="
if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution bundler --rootDir "$ZOD_TEMP_DIR" --paths '{"zod": ["'"$ZOD_TEMP_DIR"'/node_modules/zod"]}' $(find testdata/expected/ts-zod-only-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (ts-zod-only-extracted) validation PASSED"
else
    # Fallback: try with ignoreDeprecations for older TS versions
    if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution node --baseUrl "$ZOD_TEMP_DIR/node_modules" --ignoreDeprecations 6.0 $(find testdata/expected/ts-zod-only-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
        echo "✅ TypeScript (ts-zod-only-extracted) validation PASSED"
    else
        echo "❌ TypeScript (ts-zod-only-extracted) validation FAILED"
        exit 1
    fi
fi

# ============================================================================
# VALIDATE Python
# ============================================================================
echo ""
echo "=== Validating Python ==="

# Syntax check with py_compile (using find pattern like tsc)
echo "Checking Python syntax..."
for py_file in $(find testdata/expected/py -name "*.py" 2>/dev/null); do
    if ! python3 -m py_compile "$py_file" 2>/dev/null; then
        echo "❌ Python syntax validation FAILED"
        exit 1
    fi
done
echo "✅ Python syntax validation PASSED"

# Black formatting check (simple - returns exit code)
# Using --line-length 200 to avoid wrapping long Field() lines in generated code
echo "Checking Python formatting with black..."
if black --version >/dev/null 2>&1; then
    if black --check --line-length 200 testdata/expected/py; then
        echo "✅ Python formatting PASSED"
    else
        echo "❌ Python formatting FAILED (run 'black --line-length 200 testdata/expected/py' to fix)"
        exit 1
    fi
else
    echo "⚠️  black not available, skipping formatting check"
fi

echo "✅ Python validation PASSED"

# ============================================================================
# VALIDATE Go
# ============================================================================
echo ""
echo "=== Validating Go (default) ==="

echo "Checking Go syntax with gofmt..."
go_errors=$(find testdata/expected/go -name "*.go" -exec gofmt -e {} \; 2>&1 >/dev/null || true)
if [ -n "$go_errors" ]; then
    echo "$go_errors"
    echo "❌ Go syntax check FAILED"
    exit 1
fi
echo "✅ Go syntax check PASSED"

echo "Checking Go AST parsing..."
GO_PARSER=$(mktemp)
go build -o "$GO_PARSER" scripts/parse_go.go
if ! "$GO_PARSER" $(find testdata/expected/go -name "*.go"); then
    echo "❌ Go AST check FAILED"
    rm -f "$GO_PARSER"
    exit 1
fi
rm -f "$GO_PARSER"
echo "✅ Go AST check PASSED"

# ============================================================================
# SUCCESS
# ============================================================================
echo ""
echo "================================================"
echo "✅ ALL VALIDATIONS PASSED!"
echo "================================================"
echo ""
