# Makefile for building the Go WASM decoder

.PHONY: build clean wasm-exec

# Build the WASM decoder
build: wasm-exec
	@echo "Building Go WASM decoder..."
	cd ../decoder && \
	GOOS=js GOARCH=wasm go build -o ../wasm/decoder.wasm .
	@echo "✅ WASM decoder built successfully: web/scanner/wasm/decoder.wasm"

# Copy wasm_exec.js from Go toolchain
wasm-exec:
	@echo "Copying wasm_exec.js from Go toolchain..."
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" ./wasm_exec.js
	@echo "✅ wasm_exec.js copied"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f decoder.wasm wasm_exec.js
	@echo "✅ Build artifacts cleaned"

# Build with optimization flags
build-optimized: wasm-exec
	@echo "Building optimized Go WASM decoder..."
	cd ../decoder && \
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o ../wasm/decoder.wasm .
	@echo "✅ Optimized WASM decoder built: web/scanner/wasm/decoder.wasm"

# Verify the build
verify:
	@echo "Verifying WASM build..."
	@if [ -f decoder.wasm ]; then \
		echo "✅ decoder.wasm exists"; \
		ls -lh decoder.wasm; \
	else \
		echo "❌ decoder.wasm not found"; \
		exit 1; \
	fi
	@if [ -f wasm_exec.js ]; then \
		echo "✅ wasm_exec.js exists"; \
	else \
		echo "❌ wasm_exec.js not found"; \
		exit 1; \
	fi

# Development build (with debug info)
dev: build verify

# Production build (optimized)
prod: build-optimized verify

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build WASM decoder"
	@echo "  build-optimized- Build optimized WASM decoder"
	@echo "  clean          - Clean build artifacts"
	@echo "  verify         - Verify build output"
	@echo "  dev            - Development build with verification"
	@echo "  prod           - Production build with verification"
	@echo "  help           - Show this help"