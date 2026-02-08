#!/usr/bin/env bash
set -e

cd "$(dirname "$0")/.."

# Clean up Python cache directories before verification
find testdata/expected -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true

# Single file schemas
./schemagen verify testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts/foundation --quiet --disable-timestamp
./schemagen verify testdata/schemas/foundation/foundation.json --out-py testdata/expected/py/foundation --quiet --disable-timestamp
./schemagen verify testdata/schemas/foundation/foundation.json --out-go testdata/expected/go/foundation --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts/messaging_api --quiet --disable-timestamp
./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-py testdata/expected/py/messaging_api --quiet --disable-timestamp
./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-go testdata/expected/go/messaging_api --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts/extraction/blog_post --quiet --disable-timestamp
./schemagen verify testdata/schemas/extraction/blog_post.json --out-py testdata/expected/py/extraction/blog_post --quiet --disable-timestamp
./schemagen verify testdata/schemas/extraction/blog_post.json --out-go testdata/expected/go/extraction/blog_post --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/refs/organization.json --out-ts testdata/expected/ts/refs/organization --quiet --disable-timestamp
./schemagen verify testdata/schemas/refs/organization.json --out-py testdata/expected/py/refs/organization --quiet --disable-timestamp
./schemagen verify testdata/schemas/refs/organization.json --out-go testdata/expected/go/refs/organization --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts/complex/ecommerce_order --quiet --disable-timestamp
./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-py testdata/expected/py/complex/ecommerce_order --quiet --disable-timestamp
./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-go testdata/expected/go/complex/ecommerce_order --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/basic/ --out-ts testdata/expected/ts/basic --quiet --disable-timestamp
./schemagen verify testdata/schemas/basic/ --out-py testdata/expected/py/basic --quiet --disable-timestamp
./schemagen verify testdata/schemas/basic/ --out-go testdata/expected/go/basic --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/edge-cases/ --out-ts testdata/expected/ts/edge-cases --quiet --disable-timestamp
./schemagen verify testdata/schemas/edge-cases/ --out-py testdata/expected/py/edge-cases --quiet --disable-timestamp
./schemagen verify testdata/schemas/edge-cases/ --out-go testdata/expected/go/edge-cases --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/yaml/ --out-ts testdata/expected/ts/yaml --quiet --disable-timestamp
./schemagen verify testdata/schemas/yaml/ --out-py testdata/expected/py/yaml --quiet --disable-timestamp
./schemagen verify testdata/schemas/yaml/ --out-go testdata/expected/go/yaml --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

# Directory schemas
./schemagen verify testdata/schemas/events/ --out-ts testdata/expected/ts/events --quiet --disable-timestamp
./schemagen verify testdata/schemas/events/ --out-py testdata/expected/py/events --quiet --disable-timestamp
./schemagen verify testdata/schemas/events/ --out-go testdata/expected/go/events --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/anyof/ --out-ts testdata/expected/ts/anyof --quiet --disable-timestamp
./schemagen verify testdata/schemas/anyof/ --out-py testdata/expected/py/anyof --quiet --disable-timestamp
./schemagen verify testdata/schemas/anyof/ --out-go testdata/expected/go/anyof --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

./schemagen verify testdata/schemas/allof/ --out-ts testdata/expected/ts/allof --quiet --disable-timestamp
./schemagen verify testdata/schemas/allof/ --out-py testdata/expected/py/allof --quiet --disable-timestamp
./schemagen verify testdata/schemas/allof/ --out-go testdata/expected/go/allof --go-module-path "github.com/mirpo/schemagen/testdata/expected" --quiet --disable-timestamp

# Extraction schemas (with --extract-inline flag) - TypeScript
./schemagen verify testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts-extracted/foundation --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts-extracted/messaging_api --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts-extracted/extraction/blog_post --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/refs/organization.json --out-ts testdata/expected/ts-extracted/refs/organization --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts-extracted/complex/ecommerce_order --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/basic/ --out-ts testdata/expected/ts-extracted/basic --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/edge-cases/ --out-ts testdata/expected/ts-extracted/edge-cases --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/yaml/ --out-ts testdata/expected/ts-extracted/yaml --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/events/ --out-ts testdata/expected/ts-extracted/events --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/anyof/ --out-ts testdata/expected/ts-extracted/anyof --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/allof/ --out-ts testdata/expected/ts-extracted/allof --extract-inline --quiet --disable-timestamp

# ts-zod-extracted: ts-zod + extract-inline
./schemagen verify testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts-zod-extracted/foundation --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts-zod-extracted/messaging_api --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts-zod-extracted/extraction/blog_post --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/refs/organization.json --out-ts testdata/expected/ts-zod-extracted/refs/organization --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts-zod-extracted/complex/ecommerce_order --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/basic/ --out-ts testdata/expected/ts-zod-extracted/basic --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/edge-cases/ --out-ts testdata/expected/ts-zod-extracted/edge-cases --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/yaml/ --out-ts testdata/expected/ts-zod-extracted/yaml --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/events/ --out-ts testdata/expected/ts-zod-extracted/events --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/anyof/ --out-ts testdata/expected/ts-zod-extracted/anyof --ts-zod --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/allof/ --out-ts testdata/expected/ts-zod-extracted/allof --ts-zod --extract-inline --quiet --disable-timestamp

# ts-zod-only-extracted: zod-only + extract-inline
./schemagen verify testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts-zod-only-extracted/foundation --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts-zod-only-extracted/messaging_api --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts-zod-only-extracted/extraction/blog_post --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/refs/organization.json --out-ts testdata/expected/ts-zod-only-extracted/refs/organization --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts-zod-only-extracted/complex/ecommerce_order --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/basic/ --out-ts testdata/expected/ts-zod-only-extracted/basic --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/edge-cases/ --out-ts testdata/expected/ts-zod-only-extracted/edge-cases --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/yaml/ --out-ts testdata/expected/ts-zod-only-extracted/yaml --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/events/ --out-ts testdata/expected/ts-zod-only-extracted/events --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/anyof/ --out-ts testdata/expected/ts-zod-only-extracted/anyof --ts-zod-only --extract-inline --quiet --disable-timestamp
./schemagen verify testdata/schemas/allof/ --out-ts testdata/expected/ts-zod-only-extracted/allof --ts-zod-only --extract-inline --quiet --disable-timestamp

# Note: Go always extracts inline types (no go-extracted needed, see pkg/generation/pipeline.go)

echo "✅ All tests passed!"
