/**
 * Device Capabilities Detection
 * Detects browser and device capabilities for optimal scanner configuration
 */

class CapabilitiesDetector {
    constructor() {
        this.capabilities = {
            // Core browser support
            getUserMedia: false,
            webRTC: false,
            webAssembly: false,
            webWorkers: false,
            offscreenCanvas: false,

            // Camera features
            mediaDevices: false,
            facingMode: false,
            torch: false,
            zoom: false,
            focusMode: false,
            exposureMode: false,
            whiteBalanceMode: false,
            imageCapture: false,
            pointsOfInterest: false,

            // Video frame features
            requestVideoFrameCallback: false,
            videoTexture: false,

            // Input features
            touchEvents: false,
            pointerEvents: false,
            gestureEvents: false,

            // Performance features
            hardwareAcceleration: false,
            sharedArrayBuffer: false,
            transferableObjects: false,

            // Device info
            deviceType: 'unknown',
            platform: 'unknown',
            browserEngine: 'unknown',
            performanceTier: 'unknown'
        };

        this.deviceInfo = {
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            deviceMemory: navigator.deviceMemory || null,
            hardwareConcurrency: navigator.hardwareConcurrency || 1,
            maxTouchPoints: navigator.maxTouchPoints || 0
        };
    }

    /**
     * Detect all capabilities
     */
    async detect() {
        console.log('[CapabilitiesDetector] Starting capability detection...');

        // Core browser features
        this.detectCoreBrowserFeatures();

        // Device and platform info
        this.detectDeviceInfo();

        // Input capabilities
        this.detectInputCapabilities();

        // Camera capabilities (requires permission)
        try {
            await this.detectCameraCapabilities();
        } catch (error) {
            console.warn('[CapabilitiesDetector] Camera capability detection failed:', error.message);
        }

        // Performance tier estimation
        this.estimatePerformanceTier();

        console.log('[CapabilitiesDetector] Capability detection complete:', this.capabilities);
        return this.capabilities;
    }

    /**
     * Detect core browser features
     */
    detectCoreBrowserFeatures() {
        // getUserMedia support
        this.capabilities.getUserMedia = !!(
            navigator.mediaDevices &&
            navigator.mediaDevices.getUserMedia
        );

        // WebRTC support
        this.capabilities.webRTC = !!(
            window.RTCPeerConnection ||
            window.webkitRTCPeerConnection ||
            window.mozRTCPeerConnection
        );

        // WebAssembly support
        this.capabilities.webAssembly = typeof WebAssembly === 'object';

        // Web Workers support
        this.capabilities.webWorkers = typeof Worker !== 'undefined';

        // OffscreenCanvas support
        this.capabilities.offscreenCanvas = typeof OffscreenCanvas !== 'undefined';

        // MediaDevices API
        this.capabilities.mediaDevices = !!(navigator.mediaDevices);

        // ImageCapture API
        this.capabilities.imageCapture = typeof ImageCapture !== 'undefined';

        // requestVideoFrameCallback
        this.capabilities.requestVideoFrameCallback = !!(
            HTMLVideoElement.prototype.requestVideoFrameCallback
        );

        // SharedArrayBuffer (for WASM threading)
        this.capabilities.sharedArrayBuffer = typeof SharedArrayBuffer !== 'undefined';

        // Transferable objects
        this.capabilities.transferableObjects = this.testTransferableObjects();

        // Browser engine detection
        this.capabilities.browserEngine = this.detectBrowserEngine();
    }

    /**
     * Detect device and platform info
     */
    detectDeviceInfo() {
        const ua = navigator.userAgent.toLowerCase();

        // Device type
        if (/mobile|android|iphone|ipod|blackberry|iemobile|opera mini/i.test(ua)) {
            this.capabilities.deviceType = 'mobile';
        } else if (/tablet|ipad/i.test(ua)) {
            this.capabilities.deviceType = 'tablet';
        } else {
            this.capabilities.deviceType = 'desktop';
        }

        // Platform
        if (/android/i.test(ua)) {
            this.capabilities.platform = 'android';
        } else if (/ios|iphone|ipad|ipod/i.test(ua)) {
            this.capabilities.platform = 'ios';
        } else if (/windows/i.test(ua)) {
            this.capabilities.platform = 'windows';
        } else if (/mac/i.test(ua)) {
            this.capabilities.platform = 'macos';
        } else if (/linux/i.test(ua)) {
            this.capabilities.platform = 'linux';
        }

        // Hardware acceleration (indirect detection)
        this.capabilities.hardwareAcceleration = this.detectHardwareAcceleration();
    }

    /**
     * Detect input capabilities
     */
    detectInputCapabilities() {
        // Touch events
        this.capabilities.touchEvents = 'ontouchstart' in window ||
                                       navigator.maxTouchPoints > 0;

        // Pointer events
        this.capabilities.pointerEvents = window.PointerEvent !== undefined;

        // Gesture events (iOS Safari)
        this.capabilities.gestureEvents = 'ongesturestart' in window;
    }

    /**
     * Detect camera capabilities
     */
    async detectCameraCapabilities() {
        if (!this.capabilities.getUserMedia) {
            return;
        }

        try {
            // Get available devices
            const devices = await navigator.mediaDevices.enumerateDevices();
            const videoInputs = devices.filter(device => device.kind === 'videoinput');

            if (videoInputs.length === 0) {
                return;
            }

            // Test basic camera access
            const constraints = {
                video: {
                    facingMode: 'environment',
                    width: { ideal: 640 },
                    height: { ideal: 480 }
                }
            };

            const stream = await navigator.mediaDevices.getUserMedia(constraints);
            const track = stream.getVideoTracks()[0];

            if (track) {
                const capabilities = track.getCapabilities();
                const settings = track.getSettings();

                // Check specific capabilities
                this.capabilities.facingMode = 'facingMode' in capabilities;
                this.capabilities.torch = 'torch' in capabilities;
                this.capabilities.zoom = 'zoom' in capabilities;
                this.capabilities.focusMode = 'focusMode' in capabilities;
                this.capabilities.exposureMode = 'exposureMode' in capabilities;
                this.capabilities.whiteBalanceMode = 'whiteBalanceMode' in capabilities;
                this.capabilities.pointsOfInterest = 'pointsOfInterest' in capabilities;

                console.log('[CapabilitiesDetector] Camera capabilities:', capabilities);
                console.log('[CapabilitiesDetector] Camera settings:', settings);
            }

            // Clean up
            stream.getTracks().forEach(track => track.stop());

        } catch (error) {
            console.warn('[CapabilitiesDetector] Camera capabilities detection failed:', error);
        }
    }

    /**
     * Detect browser engine
     */
    detectBrowserEngine() {
        const ua = navigator.userAgent;

        if (ua.includes('Chrome')) {
            return 'blink';
        } else if (ua.includes('Firefox')) {
            return 'gecko';
        } else if (ua.includes('Safari') && !ua.includes('Chrome')) {
            return 'webkit';
        } else if (ua.includes('Edge')) {
            return 'edgehtml';
        } else {
            return 'unknown';
        }
    }

    /**
     * Test transferable objects support
     */
    testTransferableObjects() {
        try {
            const buffer = new ArrayBuffer(1);
            const worker = new Worker('data:text/javascript,');
            worker.postMessage(buffer, [buffer]);
            worker.terminate();
            return buffer.byteLength === 0; // Should be transferred
        } catch (error) {
            return false;
        }
    }

    /**
     * Detect hardware acceleration
     */
    detectHardwareAcceleration() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

            if (!gl) {
                return false;
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                return !renderer.includes('SwiftShader') && !renderer.includes('Software');
            }

            return true; // Assume hardware acceleration if we can't detect software rendering
        } catch (error) {
            return false;
        }
    }

    /**
     * Estimate performance tier
     */
    estimatePerformanceTier() {
        let score = 0;

        // Device memory
        if (this.deviceInfo.deviceMemory) {
            score += Math.min(this.deviceInfo.deviceMemory, 8);
        } else {
            score += 2; // Default assumption
        }

        // Hardware concurrency
        score += Math.min(this.deviceInfo.hardwareConcurrency, 8);

        // Device type bonus/penalty
        if (this.capabilities.deviceType === 'desktop') {
            score += 4;
        } else if (this.capabilities.deviceType === 'tablet') {
            score += 2;
        }

        // Hardware acceleration bonus
        if (this.capabilities.hardwareAcceleration) {
            score += 3;
        }

        // Modern browser features
        if (this.capabilities.webAssembly) score += 1;
        if (this.capabilities.offscreenCanvas) score += 1;
        if (this.capabilities.sharedArrayBuffer) score += 1;
        if (this.capabilities.requestVideoFrameCallback) score += 1;

        // Classify performance tier
        if (score >= 12) {
            this.capabilities.performanceTier = 'high';
        } else if (score >= 8) {
            this.capabilities.performanceTier = 'medium';
        } else {
            this.capabilities.performanceTier = 'low';
        }

        console.log('[CapabilitiesDetector] Performance score:', score, 'tier:', this.capabilities.performanceTier);
    }

    /**
     * Get recommended scanner configuration
     */
    getRecommendedConfig() {
        const config = {
            // Frame processing
            maxFrameRate: 30,
            frameWidth: 1280,
            frameHeight: 720,

            // WASM worker
            useWebWorker: this.capabilities.webWorkers,
            useSharedArrayBuffer: this.capabilities.sharedArrayBuffer,

            // Camera settings
            preferredFacingMode: 'environment',
            enableTorch: this.capabilities.torch,
            enableZoom: this.capabilities.zoom,
            enableTapToFocus: this.capabilities.pointsOfInterest || this.capabilities.focusMode,

            // Performance optimizations
            useRequestVideoFrameCallback: this.capabilities.requestVideoFrameCallback,
            useOffscreenCanvas: this.capabilities.offscreenCanvas,

            // UI adaptations
            enableTouchGestures: this.capabilities.touchEvents,
            enablePointerEvents: this.capabilities.pointerEvents,
            showAdvancedControls: this.capabilities.deviceType === 'desktop'
        };

        // Performance tier adjustments
        switch (this.capabilities.performanceTier) {
            case 'high':
                config.maxFrameRate = 30;
                config.frameWidth = 1920;
                config.frameHeight = 1080;
                break;
            case 'medium':
                config.maxFrameRate = 24;
                config.frameWidth = 1280;
                config.frameHeight = 720;
                break;
            case 'low':
                config.maxFrameRate = 15;
                config.frameWidth = 960;
                config.frameHeight = 540;
                break;
        }

        // Platform-specific adjustments
        if (this.capabilities.platform === 'ios') {
            // iOS Safari has some limitations
            config.useSharedArrayBuffer = false;
            config.maxFrameRate = Math.min(config.maxFrameRate, 24);
        }

        return config;
    }

    /**
     * Get capability summary
     */
    getSummary() {
        return {
            capabilities: { ...this.capabilities },
            deviceInfo: { ...this.deviceInfo },
            recommendedConfig: this.getRecommendedConfig()
        };
    }

    /**
     * Check if feature is supported
     */
    isSupported(feature) {
        return this.capabilities[feature] === true;
    }

    /**
     * Check if scanner is supported
     */
    isScannerSupported() {
        return this.capabilities.getUserMedia &&
               this.capabilities.webAssembly &&
               this.capabilities.webWorkers;
    }
}

// Make available globally and as module
if (typeof window !== 'undefined') {
    window.CapabilitiesDetector = CapabilitiesDetector;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = CapabilitiesDetector;
}