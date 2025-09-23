#!/bin/bash

# Build script for Go WASM decoder
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DECODER_DIR="$SCRIPT_DIR/../decoder"
OUTPUT_DIR="$SCRIPT_DIR"

echo "🔧 Building Go WASM decoder..."

# Ensure we're in the right directory
cd "$SCRIPT_DIR"

# Copy wasm_exec.js if it doesn't exist or is outdated
WASM_EXEC_PATH="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC_PATH" ]; then
    # Try alternative path
    WASM_EXEC_PATH="$(go env GOROOT)/misc/wasm/wasm_exec.js"
fi

if [ ! -f "wasm_exec.js" ] || [ "$WASM_EXEC_PATH" -nt "wasm_exec.js" ]; then
    echo "📦 Copying wasm_exec.js from Go toolchain..."
    cp "$WASM_EXEC_PATH" ./wasm_exec.js
    echo "✅ wasm_exec.js updated"
fi

# Build the WASM module
echo "🏗️  Compiling Go to WASM..."
cd "$DECODER_DIR"

# Use optimization flags for production builds
if [ "$1" = "prod" ] || [ "$1" = "production" ]; then
    echo "🚀 Building production version (optimized)..."
    GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUTPUT_DIR/decoder.wasm" .
else
    echo "🛠️  Building development version..."
    GOOS=js GOARCH=wasm go build -o "$OUTPUT_DIR/decoder.wasm" .
fi

cd "$OUTPUT_DIR"

# Verify the build
if [ -f "decoder.wasm" ]; then
    SIZE=$(du -h decoder.wasm | cut -f1)
    echo "✅ WASM decoder built successfully!"
    echo "📦 Size: $SIZE"
    echo "📍 Location: web/scanner/wasm/decoder.wasm"
else
    echo "❌ Build failed - decoder.wasm not found"
    exit 1
fi

# Optional: Run basic verification
if command -v file >/dev/null 2>&1; then
    FILE_TYPE=$(file decoder.wasm)
    echo "🔍 File type: $FILE_TYPE"
fi

echo "🎉 Build complete!"
echo ""
echo "Next steps:"
echo "1. Ensure the WASM files are served by your web server"
echo "2. Load the decoder in a Web Worker"
echo "3. Test with the scanner interface"