/**
 * Camera Control Module
 * Handles camera access, configuration, and frame capture
 */

class CameraManager {
    constructor(options = {}) {
        this.stream = null;
        this.video = null;
        this.canvas = null;
        this.context = null;
        this.track = null;

        this.config = {
            width: { ideal: 1920, max: 3840 },
            height: { ideal: 1080, max: 2160 },
            facingMode: 'environment', // Back camera for scanning
            frameRate: { ideal: 30, max: 60 },
            focusMode: 'continuous',
            ...options
        };

        this.capabilities = {
            torch: false,
            zoom: false,
            focusMode: false,
            exposure: false,
            pointsOfInterest: false
        };

        this.settings = {
            torch: false,
            zoom: 1.0,
            focusDistance: null,
            exposureCompensation: null
        };

        this.eventListeners = {
            frame: [],
            error: [],
            started: [],
            stopped: []
        };

        this.frameCallbacks = [];
        this.isCapturing = false;
        this.frameRequestId = null;
    }

    /**
     * Initialize camera access
     */
    async initialize(videoElement) {
        try {
            this.video = videoElement;

            // Request camera access with optimal settings
            const constraints = {
                video: {
                    ...this.config,
                    // Additional constraints for better barcode scanning
                    whiteBalanceMode: 'auto',
                    exposureMode: 'auto',
                    focusMode: this.config.focusMode
                },
                audio: false
            };

            console.log('[CameraManager] Requesting camera access with constraints:', constraints);

            this.stream = await navigator.mediaDevices.getUserMedia(constraints);
            this.video.srcObject = this.stream;

            // Get the video track for advanced controls
            this.track = this.stream.getVideoTracks()[0];

            // Detect capabilities
            await this.detectCapabilities();

            // Set up frame capture
            this.setupFrameCapture();

            // Wait for video to be ready
            await new Promise((resolve, reject) => {
                this.video.addEventListener('loadedmetadata', resolve);
                this.video.addEventListener('error', reject);
                setTimeout(() => reject(new Error('Video load timeout')), 5000);
            });

            console.log('[CameraManager] Camera initialized successfully');
            console.log('[CameraManager] Video dimensions:', this.video.videoWidth, 'x', this.video.videoHeight);
            console.log('[CameraManager] Capabilities:', this.capabilities);

            this.emit('started', {
                width: this.video.videoWidth,
                height: this.video.videoHeight,
                capabilities: this.capabilities
            });

            return true;

        } catch (error) {
            console.error('[CameraManager] Failed to initialize camera:', error);
            this.emit('error', {
                error: 'CAMERA_INIT_FAILED',
                message: error.message,
                details: error
            });
            throw error;
        }
    }

    /**
     * Detect camera capabilities
     */
    async detectCapabilities() {
        if (!this.track) {
            return;
        }

        try {
            const capabilities = this.track.getCapabilities();
            const settings = this.track.getSettings();

            console.log('[CameraManager] Track capabilities:', capabilities);
            console.log('[CameraManager] Track settings:', settings);

            // Check for torch support
            this.capabilities.torch = 'torch' in capabilities;

            // Check for zoom support
            this.capabilities.zoom = 'zoom' in capabilities;
            if (this.capabilities.zoom) {
                this.capabilities.zoomRange = {
                    min: capabilities.zoom?.min || 1,
                    max: capabilities.zoom?.max || 10,
                    step: capabilities.zoom?.step || 0.1
                };
            }

            // Check for focus mode support
            this.capabilities.focusMode = 'focusMode' in capabilities;
            if (this.capabilities.focusMode) {
                this.capabilities.focusModes = capabilities.focusMode || [];
            }

            // Check for exposure support
            this.capabilities.exposure = 'exposureCompensation' in capabilities;
            if (this.capabilities.exposure) {
                this.capabilities.exposureRange = {
                    min: capabilities.exposureCompensation?.min || -3,
                    max: capabilities.exposureCompensation?.max || 3,
                    step: capabilities.exposureCompensation?.step || 0.33
                };
            }

            // Check for points of interest support (tap-to-focus)
            this.capabilities.pointsOfInterest = 'pointsOfInterest' in capabilities;

        } catch (error) {
            console.warn('[CameraManager] Failed to detect capabilities:', error);
        }
    }

    /**
     * Set up frame capture canvas
     */
    setupFrameCapture() {
        this.canvas = document.createElement('canvas');
        this.context = this.canvas.getContext('2d');

        // Configure canvas for optimal performance
        this.context.imageSmoothingEnabled = false;
        this.context.imageSmoothingQuality = 'high';
    }

    /**
     * Start frame capture
     */
    startFrameCapture(callback, fps = 30) {
        if (this.isCapturing) {
            console.warn('[CameraManager] Frame capture already running');
            return;
        }

        this.frameCallbacks.push(callback);
        this.isCapturing = true;

        const frameInterval = 1000 / fps;
        let lastFrameTime = 0;

        const captureFrame = (currentTime) => {
            if (!this.isCapturing) {
                return;
            }

            // Throttle frame rate
            if (currentTime - lastFrameTime >= frameInterval) {
                this.captureFrame();
                lastFrameTime = currentTime;
            }

            this.frameRequestId = requestAnimationFrame(captureFrame);
        };

        this.frameRequestId = requestAnimationFrame(captureFrame);
        console.log('[CameraManager] Frame capture started at', fps, 'fps');
    }

    /**
     * Stop frame capture
     */
    stopFrameCapture() {
        this.isCapturing = false;
        this.frameCallbacks = [];

        if (this.frameRequestId) {
            cancelAnimationFrame(this.frameRequestId);
            this.frameRequestId = null;
        }

        console.log('[CameraManager] Frame capture stopped');
    }

    /**
     * Capture a single frame
     */
    captureFrame() {
        if (!this.video || !this.canvas || !this.context) {
            return null;
        }

        const width = this.video.videoWidth;
        const height = this.video.videoHeight;

        if (width === 0 || height === 0) {
            return null;
        }

        // Resize canvas if needed
        if (this.canvas.width !== width || this.canvas.height !== height) {
            this.canvas.width = width;
            this.canvas.height = height;
        }

        // Draw video frame to canvas
        this.context.drawImage(this.video, 0, 0, width, height);

        // Get image data
        const imageData = this.context.getImageData(0, 0, width, height);

        // Notify callbacks
        this.frameCallbacks.forEach(callback => {
            try {
                callback(imageData, width, height);
            } catch (error) {
                console.error('[CameraManager] Frame callback error:', error);
            }
        });

        // Emit frame event
        this.emit('frame', { imageData, width, height });

        return imageData;
    }

    /**
     * Toggle torch (flashlight)
     */
    async setTorch(enabled) {
        if (!this.capabilities.torch) {
            throw new Error('Torch not supported');
        }

        try {
            await this.track.applyConstraints({
                advanced: [{ torch: enabled }]
            });

            this.settings.torch = enabled;
            console.log('[CameraManager] Torch', enabled ? 'enabled' : 'disabled');

            return true;

        } catch (error) {
            console.error('[CameraManager] Failed to set torch:', error);
            throw error;
        }
    }

    /**
     * Set zoom level
     */
    async setZoom(zoomLevel) {
        if (!this.capabilities.zoom) {
            throw new Error('Zoom not supported');
        }

        const { min, max } = this.capabilities.zoomRange;
        const clampedZoom = Math.max(min, Math.min(max, zoomLevel));

        try {
            await this.track.applyConstraints({
                advanced: [{ zoom: clampedZoom }]
            });

            this.settings.zoom = clampedZoom;
            console.log('[CameraManager] Zoom set to', clampedZoom);

            return clampedZoom;

        } catch (error) {
            console.error('[CameraManager] Failed to set zoom:', error);
            throw error;
        }
    }

    /**
     * Set focus mode
     */
    async setFocusMode(mode) {
        if (!this.capabilities.focusMode) {
            throw new Error('Focus mode not supported');
        }

        if (!this.capabilities.focusModes.includes(mode)) {
            throw new Error(`Focus mode '${mode}' not supported`);
        }

        try {
            await this.track.applyConstraints({
                advanced: [{ focusMode: mode }]
            });

            console.log('[CameraManager] Focus mode set to', mode);
            return true;

        } catch (error) {
            console.error('[CameraManager] Failed to set focus mode:', error);
            throw error;
        }
    }

    /**
     * Set focus point (tap-to-focus)
     */
    async setFocusPoint(x, y) {
        if (!this.capabilities.pointsOfInterest) {
            console.warn('[CameraManager] Points of interest not supported, trying manual focus');
            // Fallback: try to adjust focus and exposure
            return this.setManualFocus(x, y);
        }

        try {
            await this.track.applyConstraints({
                advanced: [{
                    pointsOfInterest: [{ x, y }]
                }]
            });

            console.log('[CameraManager] Focus point set to', x, y);
            return true;

        } catch (error) {
            console.error('[CameraManager] Failed to set focus point:', error);
            // Try manual focus fallback
            return this.setManualFocus(x, y);
        }
    }

    /**
     * Manual focus simulation (fallback)
     */
    async setManualFocus(x, y) {
        try {
            // Try to adjust exposure based on tap location
            if (this.capabilities.exposure) {
                // Simple exposure adjustment based on position
                const exposureValue = (x - 0.5) * 2; // -1 to 1 range
                const clampedExposure = Math.max(
                    this.capabilities.exposureRange.min,
                    Math.min(this.capabilities.exposureRange.max, exposureValue)
                );

                await this.track.applyConstraints({
                    advanced: [{ exposureCompensation: clampedExposure }]
                });

                this.settings.exposureCompensation = clampedExposure;
                console.log('[CameraManager] Manual exposure adjustment:', clampedExposure);
            }

            return true;

        } catch (error) {
            console.error('[CameraManager] Manual focus failed:', error);
            return false;
        }
    }

    /**
     * Get current camera settings
     */
    getSettings() {
        const trackSettings = this.track ? this.track.getSettings() : {};

        return {
            ...this.settings,
            trackSettings,
            videoWidth: this.video?.videoWidth || 0,
            videoHeight: this.video?.videoHeight || 0
        };
    }

    /**
     * Get camera capabilities
     */
    getCapabilities() {
        return { ...this.capabilities };
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
                console.error('[CameraManager] Event callback error:', error);
            }
        });
    }

    /**
     * Stop camera and cleanup
     */
    stop() {
        this.stopFrameCapture();

        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
            this.stream = null;
        }

        if (this.video) {
            this.video.srcObject = null;
        }

        this.track = null;
        this.emit('stopped');

        console.log('[CameraManager] Camera stopped');
    }

    /**
     * Cleanup resources
     */
    cleanup() {
        this.stop();
        this.eventListeners = {
            frame: [],
            error: [],
            started: [],
            stopped: []
        };
    }
}

// Make available globally and as module
if (typeof window !== 'undefined') {
    window.CameraManager = CameraManager;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = CameraManager;
}