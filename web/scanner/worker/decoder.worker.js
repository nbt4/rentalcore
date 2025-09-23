/**
 * WASM Barcode Decoder Web Worker
 * Loads and runs the Go WASM decoder in a background thread
 */

let wasmModule = null;
let wasmReady = false;

// Configuration
const WORKER_CONFIG = {
    wasmPath: '/static/scanner/wasm/decoder.wasm',
    wasmExecPath: '/static/scanner/wasm/wasm_exec.js',
    timeout: 5000, // 5 second timeout for WASM loading
    maxRetries: 3
};

/**
 * Load the WASM execution environment
 */
async function loadWasmExec() {
    try {
        // Import the Go WASM execution environment
        importScripts(WORKER_CONFIG.wasmExecPath);

        if (typeof Go === 'undefined') {
            throw new Error('Go WASM runtime not loaded');
        }

        return true;
    } catch (error) {
        console.error('[DecoderWorker] Failed to load wasm_exec.js:', error);
        return false;
    }
}

/**
 * Initialize the WASM module
 */
async function initializeWasm() {
    try {
        console.log('[DecoderWorker] Initializing WASM decoder...');

        // Load the Go WASM execution environment
        if (!await loadWasmExec()) {
            throw new Error('Failed to load WASM execution environment');
        }

        // Create Go instance
        const go = new Go();

        // Load the WASM module
        const response = await fetch(WORKER_CONFIG.wasmPath);
        if (!response.ok) {
            throw new Error(`Failed to fetch WASM: ${response.status} ${response.statusText}`);
        }

        const wasmBytes = await response.arrayBuffer();
        const wasmModule = await WebAssembly.instantiate(wasmBytes, go.importObject);

        // Start the Go program
        go.run(wasmModule.instance);

        // Wait for the Go program to signal it's ready
        let readyCheckCount = 0;
        const maxReadyChecks = 100; // 5 seconds at 50ms intervals

        while (!self.goWasmReady && readyCheckCount < maxReadyChecks) {
            await new Promise(resolve => setTimeout(resolve, 50));
            readyCheckCount++;
        }

        if (!self.goWasmReady) {
            throw new Error('WASM module failed to initialize within timeout');
        }

        // Verify required functions are available
        if (typeof self.goDecode !== 'function') {
            throw new Error('Go decode function not available');
        }

        wasmReady = true;
        console.log('[DecoderWorker] WASM decoder initialized successfully');

        // Notify main thread that worker is ready
        self.postMessage({
            type: 'ready',
            payload: {
                success: true,
                message: 'WASM decoder ready'
            }
        });

        return true;

    } catch (error) {
        console.error('[DecoderWorker] WASM initialization failed:', error);

        self.postMessage({
            type: 'error',
            payload: {
                error: 'WASM_INIT_FAILED',
                message: error.message,
                details: error.stack
            }
        });

        return false;
    }
}

/**
 * Process a decode request
 */
function processDecodeRequest(requestId, frameData, width, height, options = {}) {
    if (!wasmReady) {
        self.postMessage({
            type: 'decode_result',
            payload: {
                requestId,
                success: false,
                error: 'WASM_NOT_READY',
                message: 'WASM decoder not initialized'
            }
        });
        return;
    }

    try {
        const startTime = performance.now();

        // Prepare parameters for Go function
        const roi = options.roi || null;
        const priority = options.priority || 0; // 0 = auto, 1 = 1D, 2 = 2D

        // Call the Go decode function
        const result = self.goDecode(frameData, width, height, roi, priority);

        const endTime = performance.now();
        const processingTime = endTime - startTime;

        // Send result back to main thread
        self.postMessage({
            type: 'decode_result',
            payload: {
                requestId,
                success: result.success,
                result: result.result || null,
                error: result.error || null,
                duplicate: result.duplicate || false,
                processingTime,
                timestamp: Date.now()
            }
        });

    } catch (error) {
        console.error('[DecoderWorker] Decode error:', error);

        self.postMessage({
            type: 'decode_result',
            payload: {
                requestId,
                success: false,
                error: 'DECODE_FAILED',
                message: error.message,
                details: error.stack
            }
        });
    }
}

/**
 * Get cache statistics
 */
function getCacheStats(requestId) {
    if (!wasmReady) {
        self.postMessage({
            type: 'cache_stats',
            payload: {
                requestId,
                success: false,
                error: 'WASM_NOT_READY'
            }
        });
        return;
    }

    try {
        const stats = self.goGetCacheStats();

        self.postMessage({
            type: 'cache_stats',
            payload: {
                requestId,
                success: true,
                stats
            }
        });

    } catch (error) {
        self.postMessage({
            type: 'cache_stats',
            payload: {
                requestId,
                success: false,
                error: error.message
            }
        });
    }
}

/**
 * Clear the decode cache
 */
function clearCache(requestId) {
    if (!wasmReady) {
        self.postMessage({
            type: 'cache_cleared',
            payload: {
                requestId,
                success: false,
                error: 'WASM_NOT_READY'
            }
        });
        return;
    }

    try {
        const result = self.goClearCache();

        self.postMessage({
            type: 'cache_cleared',
            payload: {
                requestId,
                success: result.success || true
            }
        });

    } catch (error) {
        self.postMessage({
            type: 'cache_cleared',
            payload: {
                requestId,
                success: false,
                error: error.message
            }
        });
    }
}

/**
 * Message handler for communication with main thread
 */
self.addEventListener('message', function(event) {
    const { type, payload } = event.data;

    switch (type) {
        case 'init':
            // Initialize the WASM module
            initializeWasm();
            break;

        case 'decode':
            // Process decode request
            const { requestId, frameData, width, height, options } = payload;
            processDecodeRequest(requestId, frameData, width, height, options);
            break;

        case 'get_cache_stats':
            // Get cache statistics
            getCacheStats(payload.requestId);
            break;

        case 'clear_cache':
            // Clear the decode cache
            clearCache(payload.requestId);
            break;

        case 'ping':
            // Health check
            self.postMessage({
                type: 'pong',
                payload: {
                    ready: wasmReady,
                    timestamp: Date.now()
                }
            });
            break;

        default:
            console.warn('[DecoderWorker] Unknown message type:', type);
            break;
    }
});

/**
 * Error handler for unhandled worker errors
 */
self.addEventListener('error', function(event) {
    console.error('[DecoderWorker] Unhandled error:', event.error);

    self.postMessage({
        type: 'error',
        payload: {
            error: 'WORKER_ERROR',
            message: event.error?.message || 'Unknown worker error',
            filename: event.filename,
            lineno: event.lineno,
            colno: event.colno
        }
    });
});

/**
 * Unhandled rejection handler
 */
self.addEventListener('unhandledrejection', function(event) {
    console.error('[DecoderWorker] Unhandled promise rejection:', event.reason);

    self.postMessage({
        type: 'error',
        payload: {
            error: 'PROMISE_REJECTION',
            message: event.reason?.message || 'Unhandled promise rejection',
            details: event.reason
        }
    });
});

// Log worker startup
console.log('[DecoderWorker] Barcode decoder worker initialized');
console.log('[DecoderWorker] Config:', WORKER_CONFIG);