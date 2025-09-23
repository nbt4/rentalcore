/**
 * Scanner Integration Module
 * Integrates the Go-first scanner with the existing RentalCore scan endpoints
 */

class ScannerIntegration {
    constructor(options = {}) {
        this.config = {
            baseUrl: '',
            apiEndpoints: {
                scanDevice: '/scan/{jobId}/assign',
                scanCase: '/scan/{jobId}/assign-case',
                addRental: '/api/v1/jobs/{jobId}/assign-rental'
            },
            timeout: 5000,
            retryAttempts: 3,
            retryDelay: 1000,
            ...options
        };

        this.state = {
            currentJobId: null,
            scanning: false,
            lastScanTime: 0,
            scanHistory: []
        };

        this.eventListeners = {
            scanSuccess: [],
            scanError: [],
            scanDuplicate: [],
            deviceAssigned: [],
            deviceError: []
        };

        this.stats = {
            totalScans: 0,
            successfulScans: 0,
            failedScans: 0,
            duplicateScans: 0,
            averageResponseTime: 0
        };
    }

    /**
     * Initialize the integration with a job ID
     */
    initialize(jobId) {
        this.state.currentJobId = jobId;
        console.log('[ScannerIntegration] Initialized for job:', jobId);
    }

    /**
     * Process a scan result from the WASM decoder
     */
    async processScanResult(scanResult, options = {}) {
        if (!this.state.currentJobId) {
            throw new Error('No job ID set for scanning');
        }

        const { text: barcodeData, format, timestamp } = scanResult;
        const { price, customData } = options;

        // Update statistics
        this.stats.totalScans++;

        // Check for recent duplicates (client-side)
        const isDuplicate = this.checkForDuplicate(barcodeData, timestamp);
        if (isDuplicate) {
            this.stats.duplicateScans++;
            this.emit('scanDuplicate', { barcodeData, format, timestamp });
            return { success: false, duplicate: true, message: 'Duplicate scan ignored' };
        }

        // Add to scan history
        this.state.scanHistory.push({
            barcodeData,
            format,
            timestamp,
            jobId: this.state.currentJobId
        });

        // Keep history limited
        if (this.state.scanHistory.length > 100) {
            this.state.scanHistory = this.state.scanHistory.slice(-50);
        }

        try {
            // Determine scan type and process accordingly
            const scanType = this.determineScanType(barcodeData, format);
            let result;

            switch (scanType) {
                case 'device':
                    result = await this.processDeviceScan(barcodeData, price);
                    break;
                case 'case':
                    result = await this.processCaseScan(barcodeData);
                    break;
                case 'rental':
                    result = await this.processRentalScan(barcodeData, customData);
                    break;
                default:
                    result = await this.processDeviceScan(barcodeData, price); // Default to device
                    break;
            }

            // Update statistics
            if (result.success) {
                this.stats.successfulScans++;
                this.updateAverageResponseTime(result.responseTime);
            } else {
                this.stats.failedScans++;
            }

            // Emit events
            if (result.success) {
                this.emit('scanSuccess', {
                    barcodeData,
                    format,
                    result,
                    scanType,
                    timestamp
                });

                if (scanType === 'device') {
                    this.emit('deviceAssigned', {
                        deviceId: barcodeData,
                        jobId: this.state.currentJobId,
                        result
                    });
                }
            } else {
                this.emit('scanError', {
                    barcodeData,
                    format,
                    error: result.error,
                    message: result.message,
                    scanType,
                    timestamp
                });

                if (scanType === 'device') {
                    this.emit('deviceError', {
                        deviceId: barcodeData,
                        jobId: this.state.currentJobId,
                        error: result.error,
                        message: result.message
                    });
                }
            }

            return result;

        } catch (error) {
            console.error('[ScannerIntegration] Scan processing failed:', error);
            this.stats.failedScans++;

            const errorResult = {
                success: false,
                error: 'PROCESSING_FAILED',
                message: error.message || 'Failed to process scan',
                details: error
            };

            this.emit('scanError', {
                barcodeData,
                format,
                error: errorResult.error,
                message: errorResult.message,
                timestamp
            });

            return errorResult;
        }
    }

    /**
     * Process device scan
     */
    async processDeviceScan(deviceId, price = null) {
        const startTime = performance.now();

        try {
            const url = this.config.apiEndpoints.scanDevice.replace('{jobId}', this.state.currentJobId);

            const requestData = {
                job_id: parseInt(this.state.currentJobId),
                device_id: deviceId
            };

            // Add price if provided
            if (price !== null && price !== undefined) {
                requestData.price = parseFloat(price);
            }

            console.log('[ScannerIntegration] Scanning device:', deviceId, 'for job:', this.state.currentJobId);

            const response = await this.makeRequest('POST', url, requestData);

            const responseTime = performance.now() - startTime;

            if (response.ok) {
                const result = await response.json();
                console.log('[ScannerIntegration] Device scan successful:', result);

                return {
                    success: true,
                    result,
                    responseTime,
                    scanType: 'device',
                    deviceId,
                    message: result.message || 'Device assigned successfully'
                };
            } else {
                const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
                console.error('[ScannerIntegration] Device scan failed:', response.status, errorData);

                return {
                    success: false,
                    error: 'DEVICE_SCAN_FAILED',
                    message: errorData.error || `HTTP ${response.status}`,
                    statusCode: response.status,
                    responseTime: performance.now() - startTime
                };
            }

        } catch (error) {
            console.error('[ScannerIntegration] Device scan request failed:', error);
            return {
                success: false,
                error: 'REQUEST_FAILED',
                message: error.message || 'Network request failed',
                responseTime: performance.now() - startTime
            };
        }
    }

    /**
     * Process case scan
     */
    async processCaseScan(caseId) {
        const startTime = performance.now();

        try {
            const url = this.config.apiEndpoints.scanCase.replace('{jobId}', this.state.currentJobId);

            const requestData = {
                job_id: parseInt(this.state.currentJobId),
                case_id: parseInt(caseId)
            };

            console.log('[ScannerIntegration] Scanning case:', caseId, 'for job:', this.state.currentJobId);

            const response = await this.makeRequest('POST', url, requestData);

            if (response.ok) {
                const result = await response.json();
                console.log('[ScannerIntegration] Case scan successful:', result);

                return {
                    success: true,
                    result,
                    responseTime: performance.now() - startTime,
                    scanType: 'case',
                    caseId,
                    message: result.message || 'Case devices assigned successfully'
                };
            } else {
                const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
                console.error('[ScannerIntegration] Case scan failed:', response.status, errorData);

                return {
                    success: false,
                    error: 'CASE_SCAN_FAILED',
                    message: errorData.error || `HTTP ${response.status}`,
                    statusCode: response.status,
                    responseTime: performance.now() - startTime
                };
            }

        } catch (error) {
            console.error('[ScannerIntegration] Case scan request failed:', error);
            return {
                success: false,
                error: 'REQUEST_FAILED',
                message: error.message || 'Network request failed',
                responseTime: performance.now() - startTime
            };
        }
    }

    /**
     * Process rental equipment scan
     */
    async processRentalScan(rentalId, customData = {}) {
        const startTime = performance.now();

        try {
            const url = this.config.apiEndpoints.addRental.replace('{jobId}', this.state.currentJobId);

            const requestData = {
                equipment_id: parseInt(rentalId),
                quantity: customData.quantity || 1,
                days_used: customData.daysUsed || 1,
                notes: customData.notes || ''
            };

            console.log('[ScannerIntegration] Scanning rental:', rentalId, 'for job:', this.state.currentJobId);

            const response = await this.makeRequest('POST', url, requestData);

            if (response.ok) {
                const result = await response.json();
                console.log('[ScannerIntegration] Rental scan successful:', result);

                return {
                    success: true,
                    result,
                    responseTime: performance.now() - startTime,
                    scanType: 'rental',
                    rentalId,
                    message: result.message || 'Rental equipment added successfully'
                };
            } else {
                const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
                console.error('[ScannerIntegration] Rental scan failed:', response.status, errorData);

                return {
                    success: false,
                    error: 'RENTAL_SCAN_FAILED',
                    message: errorData.error || `HTTP ${response.status}`,
                    statusCode: response.status,
                    responseTime: performance.now() - startTime
                };
            }

        } catch (error) {
            console.error('[ScannerIntegration] Rental scan request failed:', error);
            return {
                success: false,
                error: 'REQUEST_FAILED',
                message: error.message || 'Network request failed',
                responseTime: performance.now() - startTime
            };
        }
    }

    /**
     * Determine scan type based on barcode data
     */
    determineScanType(barcodeData, format) {
        // Simple heuristics to determine scan type
        // This can be enhanced based on business logic

        // Check if it's a numeric case ID (for cases)
        if (/^CASE[\d]+$/i.test(barcodeData)) {
            return 'case';
        }

        // Check if it's a rental equipment ID
        if (/^RENTAL[\d]+$/i.test(barcodeData)) {
            return 'rental';
        }

        // Default to device
        return 'device';
    }

    /**
     * Check for duplicate scans
     */
    checkForDuplicate(barcodeData, timestamp) {
        const duplicateWindow = 2000; // 2 seconds

        return this.state.scanHistory.some(scan =>
            scan.barcodeData === barcodeData &&
            (timestamp - scan.timestamp) < duplicateWindow
        );
    }

    /**
     * Make HTTP request with retry logic
     */
    async makeRequest(method, url, data = null, attempt = 1) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest'
            }
        };

        if (data && (method === 'POST' || method === 'PUT')) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(this.config.baseUrl + url, options);
            return response;

        } catch (error) {
            console.error(`[ScannerIntegration] Request failed (attempt ${attempt}):`, error);

            // Retry logic
            if (attempt < this.config.retryAttempts) {
                console.log(`[ScannerIntegration] Retrying request in ${this.config.retryDelay}ms...`);
                await new Promise(resolve => setTimeout(resolve, this.config.retryDelay));
                return this.makeRequest(method, url, data, attempt + 1);
            }

            throw error;
        }
    }

    /**
     * Update average response time
     */
    updateAverageResponseTime(newTime) {
        const { successfulScans, averageResponseTime } = this.stats;
        this.stats.averageResponseTime =
            ((averageResponseTime * (successfulScans - 1)) + newTime) / successfulScans;
    }

    /**
     * Get integration statistics
     */
    getStats() {
        return {
            ...this.stats,
            currentJobId: this.state.currentJobId,
            scanHistorySize: this.state.scanHistory.length,
            isActive: this.state.scanning
        };
    }

    /**
     * Get recent scan history
     */
    getScanHistory(limit = 10) {
        return this.state.scanHistory.slice(-limit).reverse();
    }

    /**
     * Clear scan history
     */
    clearHistory() {
        this.state.scanHistory = [];
    }

    /**
     * Set scanning state
     */
    setScanning(scanning) {
        this.state.scanning = scanning;
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
                console.error('[ScannerIntegration] Event callback error:', error);
            }
        });
    }

    /**
     * Update configuration
     */
    updateConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
    }

    /**
     * Cleanup
     */
    cleanup() {
        this.state.scanHistory = [];
        this.eventListeners = {
            scanSuccess: [],
            scanError: [],
            scanDuplicate: [],
            deviceAssigned: [],
            deviceError: []
        };
    }
}

// Make available globally and as module
if (typeof window !== 'undefined') {
    window.ScannerIntegration = ScannerIntegration;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = ScannerIntegration;
}