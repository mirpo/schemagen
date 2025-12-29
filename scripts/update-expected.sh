#!/usr/bin/env bash
set -e

cd "$(dirname "$0")/.."

echo "=== Generating DEFAULT mode outputs ==="

# Single file schemas
./schemagen generate testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts/foundation --disable-timestamp
./schemagen generate testdata/schemas/foundation/foundation.json --out-py testdata/expected/py/foundation --disable-timestamp
./schemagen generate testdata/schemas/foundation/foundation.json --out-go testdata/expected/go/foundation --go-module-path "github.com/mirpo/schemagen/testdata/expected"  --disable-timestamp

./schemagen generate testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts/messaging_api --disable-timestamp
./schemagen generate testdata/schemas/messaging_api/messaging_api.json --out-py testdata/expected/py/messaging_api --disable-timestamp
./schemagen generate testdata/schemas/messaging_api/messaging_api.json --out-go testdata/expected/go/messaging_api --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts/extraction/blog_post --disable-timestamp
./schemagen generate testdata/schemas/extraction/blog_post.json --out-py testdata/expected/py/extraction/blog_post --disable-timestamp
./schemagen generate testdata/schemas/extraction/blog_post.json --out-go testdata/expected/go/extraction/blog_post --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/refs/organization.json --out-ts testdata/expected/ts/refs/organization --disable-timestamp
./schemagen generate testdata/schemas/refs/organization.json --out-py testdata/expected/py/refs/organization --disable-timestamp
./schemagen generate testdata/schemas/refs/organization.json --out-go testdata/expected/go/refs/organization --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts/complex/ecommerce_order --disable-timestamp
./schemagen generate testdata/schemas/complex/ecommerce_order.json --out-py testdata/expected/py/complex/ecommerce_order --disable-timestamp
./schemagen generate testdata/schemas/complex/ecommerce_order.json --out-go testdata/expected/go/complex/ecommerce_order --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/basic/ --out-ts testdata/expected/ts/basic --disable-timestamp
./schemagen generate testdata/schemas/basic/ --out-py testdata/expected/py/basic --disable-timestamp
./schemagen generate testdata/schemas/basic/ --out-go testdata/expected/go/basic --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/edge-cases/ --out-ts testdata/expected/ts/edge-cases --disable-timestamp
./schemagen generate testdata/schemas/edge-cases/ --out-py testdata/expected/py/edge-cases --disable-timestamp
./schemagen generate testdata/schemas/edge-cases/ --out-go testdata/expected/go/edge-cases --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/yaml/ --out-ts testdata/expected/ts/yaml --disable-timestamp
./schemagen generate testdata/schemas/yaml/ --out-py testdata/expected/py/yaml --disable-timestamp
./schemagen generate testdata/schemas/yaml/ --out-go testdata/expected/go/yaml --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

# Directory schemas
./schemagen generate testdata/schemas/events/ --out-ts testdata/expected/ts/events --disable-timestamp
./schemagen generate testdata/schemas/events/ --out-py testdata/expected/py/events --disable-timestamp
./schemagen generate testdata/schemas/events/ --out-go testdata/expected/go/events --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/anyof/ --out-ts testdata/expected/ts/anyof --disable-timestamp
./schemagen generate testdata/schemas/anyof/ --out-py testdata/expected/py/anyof --disable-timestamp
./schemagen generate testdata/schemas/anyof/ --out-go testdata/expected/go/anyof --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

./schemagen generate testdata/schemas/allof/ --out-ts testdata/expected/ts/allof --disable-timestamp
./schemagen generate testdata/schemas/allof/ --out-py testdata/expected/py/allof --disable-timestamp
./schemagen generate testdata/schemas/allof/ --out-go testdata/expected/go/allof --go-module-path "github.com/mirpo/schemagen/testdata/expected" --disable-timestamp

echo "=== Generating EXTRACTED mode outputs (--extract-inline) ==="

# Single file schemas - extracted (TypeScript only)
./schemagen generate testdata/schemas/foundation/foundation.json --out-ts testdata/expected/ts-extracted/foundation --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/messaging_api/messaging_api.json --out-ts testdata/expected/ts-extracted/messaging_api --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/extraction/blog_post.json --out-ts testdata/expected/ts-extracted/extraction/blog_post --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/refs/organization.json --out-ts testdata/expected/ts-extracted/refs/organization --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/complex/ecommerce_order.json --out-ts testdata/expected/ts-extracted/complex/ecommerce_order --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/basic/ --out-ts testdata/expected/ts-extracted/basic --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/edge-cases/ --out-ts testdata/expected/ts-extracted/edge-cases --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/yaml/ --out-ts testdata/expected/ts-extracted/yaml --extract-inline --disable-timestamp

# Directory schemas - extracted (TypeScript only)
./schemagen generate testdata/schemas/events/ --out-ts testdata/expected/ts-extracted/events --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/anyof/ --out-ts testdata/expected/ts-extracted/anyof --extract-inline --disable-timestamp
./schemagen generate testdata/schemas/allof/ --out-ts testdata/expected/ts-extracted/allof --extract-inline --disable-timestamp

# Note: Go always extracts inline types (no go-extracted needed, see pkg/generation/pipeline.go)
