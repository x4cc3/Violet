#!/bin/bash
# Violet WASM Build Script
# Compiles the game for web browsers and creates a dist folder

set -e

echo "🎮 Building Violet for WebAssembly..."

# Create dist directory
mkdir -p dist
mkdir -p dist/assets

# Build WASM binary
echo "📦 Compiling Go to WASM..."
GOOS=js GOARCH=wasm go build -o dist/violet.wasm .

# Copy wasm_exec.js from Go installation
GOROOT=$(go env GOROOT)
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
    echo "📜 Copying wasm_exec.js..."
    cp "$GOROOT/lib/wasm/wasm_exec.js" dist/wasm_exec.js
elif [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
    # Fallback for older Go versions
    cp "$GOROOT/misc/wasm/wasm_exec.js" dist/wasm_exec.js
elif [ -f "wasm_exec.js" ]; then
    # Use local copy if available
    cp wasm_exec.js dist/wasm_exec.js
else
    echo "⚠️  Warning: Could not find wasm_exec.js. Please copy it manually from your Go installation."
fi

# Copy HTML file
echo "📄 Copying index.html..."
cp index.html dist/index.html

# Copy assets (images and sounds)
echo "🎨 Copying assets..."
cp -r assets/* dist/assets/

# Get file sizes
WASM_SIZE=$(du -h dist/violet.wasm | cut -f1)
TOTAL_SIZE=$(du -sh dist | cut -f1)

echo ""
echo "✅ Build complete!"
echo "   WASM binary: $WASM_SIZE"
echo "   Total dist:  $TOTAL_SIZE"
echo ""
echo "📂 Output directory: dist/"
echo ""
echo "To run locally:"
echo "   cd dist && python3 -m http.server 8080"
echo "   Then open http://localhost:8080"
echo ""
echo "For production, serve the dist/ folder with any web server."
