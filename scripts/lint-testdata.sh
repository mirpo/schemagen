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
if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution node --baseUrl "$ZOD_TEMP_DIR/node_modules" $(find testdata/expected/ts-zod-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (ts-zod-extracted) validation PASSED"
else
    echo "❌ TypeScript (ts-zod-extracted) validation FAILED"
    exit 1
fi

echo ""
echo "=== Validating TypeScript (ts-zod-only-extracted) ==="
if "$ZOD_TEMP_DIR/node_modules/.bin/tsc" --noEmit --skipLibCheck --moduleResolution node --baseUrl "$ZOD_TEMP_DIR/node_modules" $(find testdata/expected/ts-zod-only-extracted -name "*.ts" 2>/dev/null) 2>/dev/null; then
    echo "✅ TypeScript (ts-zod-only-extracted) validation PASSED"
else
    echo "❌ TypeScript (ts-zod-only-extracted) validation FAILED"
    exit 1
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
if command -v black >/dev/null 2>&1; then
    if black --check --line-length 200 testdata/expected/py; then
        echo "✅ Python formatting PASSED"
    else
        echo "❌ Python formatting FAILED (run 'black --line-length 200 testdata/expected/py' to fix)"
        exit 1
    fi
else
    echo "⚠️  black not installed, skipping formatting check"
fi

echo "✅ Python validation PASSED"

# ============================================================================
# VALIDATE Go
# ============================================================================
echo ""
echo "=== Validating Go (default) ==="

# Use gofmt to check syntax (doesn't require imports to be resolvable)
# Errors go to stderr, reformatted content to stdout (which we discard)
echo "Checking Go syntax with gofmt..."
go_errors=$(find testdata/expected/go -name "*.go" -exec gofmt -e {} \; 2>&1 >/dev/null || true)
if [ -n "$go_errors" ]; then
    echo "$go_errors"
    echo "❌ Go (default) syntax check FAILED"
    exit 1
fi
echo "✅ Go (default) syntax check PASSED"

# Note: Go always extracts inline types (no go-extracted needed, see pkg/generation/pipeline.go)

# ============================================================================
# SUCCESS
# ============================================================================
echo ""
echo "================================================"
echo "✅ ALL VALIDATIONS PASSED!"
echo "================================================"
echo ""
