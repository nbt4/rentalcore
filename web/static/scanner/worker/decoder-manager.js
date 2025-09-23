/**
 * Barcode Decoder Manager
 * Manages the WASM decoder worker and provides a clean API
 */

class DecoderManager {
    constructor(options = {}) {
        this.worker = null;
        this.ready = false;
        this.requestId = 0;
        this.pendingRequests = new Map();

        this.config = {
            workerPath: '/static/scanner/worker/decoder.worker.js',
            initTimeout: 10000, // 10 seconds
            decodeTimeout: 1000, // 1 second per decode
            maxRetries: 3,
            ...options
        };

        this.stats = {
            totalRequests: 0,
            successfulDecodes: 0,
            failedDecodes: 0,
            duplicates: 0,
            averageProcessingTime: 0
        };

        this.eventListeners = {
            ready: [],
            error: [],
            decode: []
        };
    }

    /**
     * Initialize the decoder worker
     */
    async initialize() {
        return new Promise((resolve, reject) => {
            try {
                // Create worker
                this.worker = new Worker(this.config.workerPath);

                // Set up message handling
                this.worker.onmessage = (event) => this.handleWorkerMessage(event);
                this.worker.onerror = (error) => this.handleWorkerError(error);

                // Set up initialization timeout
                const initTimeout = setTimeout(() => {
                    this.cleanup();
                    reject(new Error('Decoder initialization timeout'));
                }, this.config.initTimeout);

                // Listen for ready signal
                const readyHandler = () => {
                    clearTimeout(initTimeout);
                    this.ready = true;
                    this.emit('ready');
                    resolve();
                };

                this.addEventListener('ready', readyHandler);

                // Start initialization
                this.worker.postMessage({ type: 'init' });

            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Decode a frame
     */
    async decode(frameData, width, height, options = {}) {
        if (!this.ready) {
            throw new Error('Decoder not ready');
        }

        return new Promise((resolve, reject) => {
            const requestId = ++this.requestId;
            this.stats.totalRequests++;

            // Set up timeout
            const timeout = setTimeout(() => {
                this.pendingRequests.delete(requestId);
                reject(new Error('Decode timeout'));
            }, this.config.decodeTimeout);

            // Store request
            this.pendingRequests.set(requestId, {
                resolve,
                reject,
                timeout,
                startTime: performance.now()
            });

            // Send decode request
            this.worker.postMessage({
                type: 'decode',
                payload: {
                    requestId,
                    frameData,
                    width,
                    height,
                    options
                }
            });
        });
    }

    /**
     * Get cache statistics
     */
    async getCacheStats() {
        if (!this.ready) {
            throw new Error('Decoder not ready');
        }

        return new Promise((resolve, reject) => {
            const requestId = ++this.requestId;

            const timeout = setTimeout(() => {
                this.pendingRequests.delete(requestId);
                reject(new Error('Get cache stats timeout'));
            }, 1000);

            this.pendingRequests.set(requestId, {
                resolve,
                reject,
                timeout
            });

            this.worker.postMessage({
                type: 'get_cache_stats',
                payload: { requestId }
            });
        });
    }

    /**
     * Clear the decode cache
     */
    async clearCache() {
        if (!this.ready) {
            throw new Error('Decoder not ready');
        }

        return new Promise((resolve, reject) => {
            const requestId = ++this.requestId;

            const timeout = setTimeout(() => {
                this.pendingRequests.delete(requestId);
                reject(new Error('Clear cache timeout'));
            }, 1000);

            this.pendingRequests.set(requestId, {
                resolve,
                reject,
                timeout
            });

            this.worker.postMessage({
                type: 'clear_cache',
                payload: { requestId }
            });
        });
    }

    /**
     * Health check
     */
    async ping() {
        if (!this.worker) {
            return false;
        }

        return new Promise((resolve) => {
            const timeout = setTimeout(() => resolve(false), 1000);

            const pingHandler = () => {
                clearTimeout(timeout);
                resolve(true);
            };

            this.worker.onmessage = (event) => {
                if (event.data.type === 'pong') {
                    pingHandler();
                }
                this.handleWorkerMessage(event);
            };

            this.worker.postMessage({ type: 'ping' });
        });
    }

    /**
     * Handle worker messages
     */
    handleWorkerMessage(event) {
        const { type, payload } = event.data;

        switch (type) {
            case 'ready':
                this.emit('ready', payload);
                break;

            case 'decode_result':
                this.handleDecodeResult(payload);
                break;

            case 'cache_stats':
            case 'cache_cleared':
                this.handleRequestResponse(payload);
                break;

            case 'error':
                this.emit('error', payload);
                break;

            default:
                console.warn('[DecoderManager] Unknown message type:', type);
                break;
        }
    }

    /**
     * Handle decode result
     */
    handleDecodeResult(payload) {
        const { requestId, success, result, error, duplicate, processingTime } = payload;

        const request = this.pendingRequests.get(requestId);
        if (!request) {
            console.warn('[DecoderManager] No pending request for ID:', requestId);
            return;
        }

        clearTimeout(request.timeout);
        this.pendingRequests.delete(requestId);

        // Update statistics
        if (success) {
            this.stats.successfulDecodes++;
            this.updateAverageProcessingTime(processingTime);
        } else {
            this.stats.failedDecodes++;
        }

        if (duplicate) {
            this.stats.duplicates++;
        }

        // Emit decode event
        this.emit('decode', { success, result, error, duplicate, processingTime });

        // Resolve/reject promise
        if (success) {
            request.resolve({ result, duplicate, processingTime });
        } else {
            request.reject(new Error(error || 'Decode failed'));
        }
    }

    /**
     * Handle generic request response
     */
    handleRequestResponse(payload) {
        const { requestId, success, error } = payload;

        const request = this.pendingRequests.get(requestId);
        if (!request) {
            return;
        }

        clearTimeout(request.timeout);
        this.pendingRequests.delete(requestId);

        if (success) {
            request.resolve(payload);
        } else {
            request.reject(new Error(error || 'Request failed'));
        }
    }

    /**
     * Handle worker errors
     */
    handleWorkerError(error) {
        console.error('[DecoderManager] Worker error:', error);
        this.emit('error', {
            error: 'WORKER_ERROR',
            message: error.message,
            details: error
        });
    }

    /**
     * Update average processing time
     */
    updateAverageProcessingTime(newTime) {
        const { successfulDecodes, averageProcessingTime } = this.stats;
        this.stats.averageProcessingTime =
            ((averageProcessingTime * (successfulDecodes - 1)) + newTime) / successfulDecodes;
    }

    /**
     * Event management
     */
    addEventListener(event, callback) {
        if (!this.eventListeners[event]) {
            this.eventListeners[event] = [];
        }
        this.eventListeners[event].push(callback);
    }

    removeEventListener(event, callback) {
        if (!this.eventListeners[event]) {
            return;
        }
        const index = this.eventListeners[event].indexOf(callback);
        if (index > -1) {
            this.eventListeners[event].splice(index, 1);
        }
    }

    emit(event, data) {
        if (!this.eventListeners[event]) {
            return;
        }
        this.eventListeners[event].forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error('[DecoderManager] Event callback error:', error);
            }
        });
    }

    /**
     * Get decoder statistics
     */
    getStats() {
        return {
            ...this.stats,
            ready: this.ready,
            pendingRequests: this.pendingRequests.size
        };
    }

    /**
     * Cleanup resources
     */
    cleanup() {
        if (this.worker) {
            this.worker.terminate();
            this.worker = null;
        }

        // Clear pending requests
        this.pendingRequests.forEach(request => {
            clearTimeout(request.timeout);
            request.reject(new Error('Decoder terminated'));
        });
        this.pendingRequests.clear();

        this.ready = false;
        this.eventListeners = {
            ready: [],
            error: [],
            decode: []
        };
    }

    /**
     * Destroy the decoder
     */
    destroy() {
        this.cleanup();
    }
}

// Make available globally and as module
if (typeof window !== 'undefined') {
    window.DecoderManager = DecoderManager;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = DecoderManager;
}