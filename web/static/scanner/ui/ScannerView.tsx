/**
 * Scanner View Component
 * Main React component for the Go-first barcode scanner
 */

import React, { useEffect, useRef, useState, useCallback } from 'react';

interface ScannerViewProps {
    onScanResult?: (result: ScanResult) => void;
    onError?: (error: ScannerError) => void;
    onReady?: () => void;
    jobId?: string;
    enableTorch?: boolean;
    enableZoom?: boolean;
    enableTapToFocus?: boolean;
    showROIToggle?: boolean;
    className?: string;
}

interface ScanResult {
    text: string;
    format: string;
    cornerPoints: Array<{ x: number; y: number }>;
    confidence: number;
    timestamp: number;
    processingTime: number;
}

interface ScannerError {
    error: string;
    message: string;
    details?: any;
}

interface ScannerState {
    isInitializing: boolean;
    isReady: boolean;
    isScanning: boolean;
    error: string | null;
    lastResult: ScanResult | null;
    frameRate: number;
    processingTime: number;
}

interface CameraSettings {
    torch: boolean;
    zoom: number;
    focusMode: string;
}

const ScannerView: React.FC<ScannerViewProps> = ({
    onScanResult,
    onError,
    onReady,
    jobId,
    enableTorch = true,
    enableZoom = true,
    enableTapToFocus = true,
    showROIToggle = true,
    className = ''
}) => {
    // Refs
    const videoRef = useRef<HTMLVideoElement>(null);
    const overlayRef = useRef<HTMLCanvasElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    // State
    const [state, setState] = useState<ScannerState>({
        isInitializing: true,
        isReady: false,
        isScanning: false,
        error: null,
        lastResult: null,
        frameRate: 0,
        processingTime: 0
    });

    const [cameraSettings, setCameraSettings] = useState<CameraSettings>({
        torch: false,
        zoom: 1.0,
        focusMode: 'continuous'
    });

    const [capabilities, setCapabilities] = useState<any>(null);
    const [showSettings, setShowSettings] = useState(false);
    const [roiMode, setRoiMode] = useState<'auto' | '1d' | '2d'>('auto');

    // Manager instances
    const managersRef = useRef<{
        decoder?: any;
        camera?: any;
        gesture?: any;
        capabilities?: any;
    }>({});

    /**
     * Initialize scanner components
     */
    const initializeScanner = useCallback(async () => {
        try {
            setState(prev => ({ ...prev, isInitializing: true, error: null }));

            // Load required modules
            const [
                { DecoderManager },
                { CameraManager },
                { GestureManager },
                { CapabilitiesDetector }
            ] = await Promise.all([
                import('/static/scanner/worker/decoder-manager.js'),
                import('/static/scanner/ui/camera.js'),
                import('/static/scanner/ui/gestures.js'),
                import('/static/scanner/ui/capabilities.js')
            ]);

            // Detect capabilities first
            const capabilityDetector = new CapabilitiesDetector();
            const detectedCapabilities = await capabilityDetector.detect();
            setCapabilities(detectedCapabilities);

            if (!capabilityDetector.isScannerSupported()) {
                throw new Error('Scanner not supported on this device/browser');
            }

            // Initialize decoder
            const decoder = new DecoderManager();
            await decoder.initialize();

            // Initialize camera
            const camera = new CameraManager(capabilityDetector.getRecommendedConfig());
            await camera.initialize(videoRef.current!);

            // Initialize gesture manager
            const gesture = new GestureManager(containerRef.current!, {
                enablePinchZoom: enableZoom && detectedCapabilities.zoom,
                enableTapToFocus: enableTapToFocus && (detectedCapabilities.pointsOfInterest || detectedCapabilities.focusMode),
                enableDoubleTap: true
            });

            // Store managers
            managersRef.current = { decoder, camera, gesture, capabilities: capabilityDetector };

            // Set up event listeners
            setupEventListeners();

            // Start scanning
            startScanning();

            setState(prev => ({
                ...prev,
                isInitializing: false,
                isReady: true
            }));

            onReady?.();

        } catch (error) {
            console.error('[ScannerView] Initialization failed:', error);
            const scannerError: ScannerError = {
                error: 'INIT_FAILED',
                message: error instanceof Error ? error.message : 'Unknown initialization error',
                details: error
            };

            setState(prev => ({
                ...prev,
                isInitializing: false,
                error: scannerError.message
            }));

            onError?.(scannerError);
        }
    }, [enableTorch, enableZoom, enableTapToFocus, onReady, onError]);

    /**
     * Set up event listeners for all managers
     */
    const setupEventListeners = useCallback(() => {
        const { decoder, camera, gesture } = managersRef.current;

        if (!decoder || !camera || !gesture) return;

        // Decoder events
        decoder.addEventListener('decode', handleDecodeResult);
        decoder.addEventListener('error', handleDecoderError);

        // Camera events
        camera.addEventListener('frame', handleCameraFrame);
        camera.addEventListener('error', handleCameraError);

        // Gesture events
        gesture.addEventListener('pinchZoom', handleZoomGesture);
        gesture.addEventListener('tapToFocus', handleFocusGesture);
        gesture.addEventListener('doubleTap', handleDoubleTapGesture);

    }, []);

    /**
     * Handle decode result
     */
    const handleDecodeResult = useCallback((event: any) => {
        const { result, duplicate, processingTime } = event;

        if (duplicate) {
            console.log('[ScannerView] Duplicate barcode ignored');
            return;
        }

        if (result) {
            setState(prev => ({
                ...prev,
                lastResult: { ...result, processingTime },
                processingTime
            }));

            onScanResult?.(result);

            // Visual feedback
            showScanFeedback(result);
        }
    }, [onScanResult]);

    /**
     * Handle camera frame
     */
    const handleCameraFrame = useCallback((event: any) => {
        const { imageData, width, height } = event;
        const { decoder } = managersRef.current;

        if (!decoder || !state.isScanning) return;

        // Decode frame with current ROI settings
        const options = {
            priority: roiMode === '1d' ? 1 : roiMode === '2d' ? 2 : 0,
            roi: calculateROI(width, height)
        };

        decoder.decode(imageData.data, width, height, options)
            .catch((error: Error) => {
                // Ignore decode failures (expected for frames without barcodes)
                if (!error.message.includes('timeout')) {
                    console.debug('[ScannerView] Decode error:', error.message);
                }
            });
    }, [state.isScanning, roiMode]);

    /**
     * Calculate ROI based on current mode
     */
    const calculateROI = useCallback((width: number, height: number) => {
        if (roiMode === 'auto') return null;

        // Center ROI for 1D barcodes
        if (roiMode === '1d') {
            const roiWidth = Math.floor(width * 0.8);
            const roiHeight = Math.floor(height * 0.3);
            return {
                x: Math.floor((width - roiWidth) / 2),
                y: Math.floor((height - roiHeight) / 2),
                width: roiWidth,
                height: roiHeight
            };
        }

        // Full frame for 2D barcodes
        return null;
    }, [roiMode]);

    /**
     * Handle zoom gesture
     */
    const handleZoomGesture = useCallback(async (event: any) => {
        const { camera } = managersRef.current;
        if (!camera) return;

        try {
            const newZoom = await camera.setZoom(event.zoom);
            setCameraSettings(prev => ({ ...prev, zoom: newZoom }));
        } catch (error) {
            console.error('[ScannerView] Zoom failed:', error);
        }
    }, []);

    /**
     * Handle focus gesture
     */
    const handleFocusGesture = useCallback(async (event: any) => {
        const { camera } = managersRef.current;
        if (!camera) return;

        try {
            await camera.setFocusPoint(event.relative.x, event.relative.y);
        } catch (error) {
            console.error('[ScannerView] Focus failed:', error);
        }
    }, []);

    /**
     * Handle double tap gesture
     */
    const handleDoubleTapGesture = useCallback(async (event: any) => {
        const { camera } = managersRef.current;
        if (!camera) return;

        try {
            const newZoom = await camera.setZoom(event.targetZoom);
            setCameraSettings(prev => ({ ...prev, zoom: newZoom }));
        } catch (error) {
            console.error('[ScannerView] Double tap zoom failed:', error);
        }
    }, []);

    /**
     * Start scanning
     */
    const startScanning = useCallback(() => {
        const { camera } = managersRef.current;
        if (!camera) return;

        camera.startFrameCapture(null, 30); // 30 FPS
        setState(prev => ({ ...prev, isScanning: true }));
    }, []);

    /**
     * Stop scanning
     */
    const stopScanning = useCallback(() => {
        const { camera } = managersRef.current;
        if (!camera) return;

        camera.stopFrameCapture();
        setState(prev => ({ ...prev, isScanning: false }));
    }, []);

    /**
     * Toggle torch
     */
    const toggleTorch = useCallback(async () => {
        const { camera } = managersRef.current;
        if (!camera) return;

        try {
            const newTorchState = !cameraSettings.torch;
            await camera.setTorch(newTorchState);
            setCameraSettings(prev => ({ ...prev, torch: newTorchState }));
        } catch (error) {
            console.error('[ScannerView] Torch toggle failed:', error);
        }
    }, [cameraSettings.torch]);

    /**
     * Show scan feedback
     */
    const showScanFeedback = useCallback((result: ScanResult) => {
        // Visual feedback for successful scan
        if (overlayRef.current) {
            const canvas = overlayRef.current;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.clearRect(0, 0, canvas.width, canvas.height);

                // Draw corner points if available
                if (result.cornerPoints && result.cornerPoints.length > 0) {
                    ctx.strokeStyle = '#00ff00';
                    ctx.lineWidth = 3;
                    ctx.beginPath();

                    result.cornerPoints.forEach((point, index) => {
                        if (index === 0) {
                            ctx.moveTo(point.x, point.y);
                        } else {
                            ctx.lineTo(point.x, point.y);
                        }
                    });

                    ctx.closePath();
                    ctx.stroke();
                }
            }
        }

        // Audio feedback
        try {
            const audio = new Audio('data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwhBLLZ9N6VVaS1');
            audio.volume = 0.3;
            audio.play().catch(() => {}); // Ignore audio errors
        } catch (error) {
            // Ignore audio errors
        }

        // Clear feedback after delay
        setTimeout(() => {
            if (overlayRef.current) {
                const ctx = overlayRef.current.getContext('2d');
                ctx?.clearRect(0, 0, overlayRef.current.width, overlayRef.current.height);
            }
        }, 1000);
    }, []);

    /**
     * Handle errors
     */
    const handleDecoderError = useCallback((error: any) => {
        console.error('[ScannerView] Decoder error:', error);
        onError?.({
            error: 'DECODER_ERROR',
            message: error.message || 'Decoder error',
            details: error
        });
    }, [onError]);

    const handleCameraError = useCallback((error: any) => {
        console.error('[ScannerView] Camera error:', error);
        onError?.({
            error: 'CAMERA_ERROR',
            message: error.message || 'Camera error',
            details: error
        });
    }, [onError]);

    // Initialize on mount
    useEffect(() => {
        initializeScanner();

        return () => {
            // Cleanup
            const { decoder, camera, gesture } = managersRef.current;
            decoder?.cleanup();
            camera?.cleanup();
            gesture?.cleanup();
        };
    }, [initializeScanner]);

    // Update overlay canvas size when video loads
    useEffect(() => {
        const video = videoRef.current;
        const overlay = overlayRef.current;

        if (video && overlay) {
            const updateSize = () => {
                overlay.width = video.videoWidth;
                overlay.height = video.videoHeight;
                overlay.style.width = '100%';
                overlay.style.height = '100%';
            };

            video.addEventListener('loadedmetadata', updateSize);
            return () => video.removeEventListener('loadedmetadata', updateSize);
        }
    }, []);

    if (state.isInitializing) {
        return (
            <div className={`scanner-view ${className}`}>
                <div className="scanner-loading">
                    <div className="loading-spinner"></div>
                    <p>Initializing scanner...</p>
                </div>
            </div>
        );
    }

    if (state.error) {
        return (
            <div className={`scanner-view ${className}`}>
                <div className="scanner-error">
                    <h3>Scanner Error</h3>
                    <p>{state.error}</p>
                    <button onClick={initializeScanner}>Retry</button>
                </div>
            </div>
        );
    }

    return (
        <div ref={containerRef} className={`scanner-view ${className}`}>
            {/* Video preview */}
            <div className="scanner-video-container">
                <video
                    ref={videoRef}
                    autoPlay
                    playsInline
                    muted
                    className="scanner-video"
                />
                <canvas
                    ref={overlayRef}
                    className="scanner-overlay"
                />

                {/* ROI indicator */}
                {roiMode === '1d' && (
                    <div className="scanner-roi roi-1d">
                        <div className="roi-indicator"></div>
                    </div>
                )}
            </div>

            {/* Controls */}
            <div className="scanner-controls">
                {/* ROI Toggle */}
                {showROIToggle && (
                    <div className="scanner-control-group">
                        <label>Scan Mode:</label>
                        <select
                            value={roiMode}
                            onChange={(e) => setRoiMode(e.target.value as any)}
                        >
                            <option value="auto">Auto</option>
                            <option value="1d">1D Barcodes</option>
                            <option value="2d">2D Codes</option>
                        </select>
                    </div>
                )}

                {/* Torch */}
                {enableTorch && capabilities?.torch && (
                    <button
                        className={`scanner-torch ${cameraSettings.torch ? 'active' : ''}`}
                        onClick={toggleTorch}
                        title="Toggle flashlight"
                    >
                        🔦
                    </button>
                )}

                {/* Zoom */}
                {enableZoom && capabilities?.zoom && (
                    <div className="scanner-zoom">
                        <label>Zoom: {cameraSettings.zoom.toFixed(1)}x</label>
                        <input
                            type="range"
                            min={capabilities.zoomRange?.min || 1}
                            max={capabilities.zoomRange?.max || 8}
                            step={capabilities.zoomRange?.step || 0.1}
                            value={cameraSettings.zoom}
                            onChange={async (e) => {
                                const { camera } = managersRef.current;
                                if (camera) {
                                    try {
                                        const newZoom = await camera.setZoom(parseFloat(e.target.value));
                                        setCameraSettings(prev => ({ ...prev, zoom: newZoom }));
                                    } catch (error) {
                                        console.error('Zoom failed:', error);
                                    }
                                }
                            }}
                        />
                    </div>
                )}

                {/* Settings toggle */}
                <button
                    className="scanner-settings-toggle"
                    onClick={() => setShowSettings(!showSettings)}
                    title="Settings"
                >
                    ⚙️
                </button>
            </div>

            {/* Status */}
            <div className="scanner-status">
                <div className="status-indicator">
                    <span className={`status-dot ${state.isReady ? 'ready' : 'error'}`}></span>
                    <span>{state.isReady ? 'Ready' : 'Error'}</span>
                </div>

                {state.lastResult && (
                    <div className="last-scan">
                        Last: {state.lastResult.format} - {state.lastResult.text.substring(0, 20)}...
                        ({state.lastResult.processingTime}ms)
                    </div>
                )}
            </div>

            {/* Settings panel */}
            {showSettings && (
                <div className="scanner-settings">
                    <h4>Scanner Settings</h4>

                    <div className="setting-group">
                        <label>Performance:</label>
                        <span>{capabilities?.performanceTier} tier</span>
                    </div>

                    <div className="setting-group">
                        <label>Device:</label>
                        <span>{capabilities?.deviceType} ({capabilities?.platform})</span>
                    </div>

                    <div className="setting-group">
                        <label>Features:</label>
                        <div className="feature-list">
                            {capabilities?.torch && <span className="feature">Torch</span>}
                            {capabilities?.zoom && <span className="feature">Zoom</span>}
                            {capabilities?.pointsOfInterest && <span className="feature">Tap-to-Focus</span>}
                        </div>
                    </div>

                    <button onClick={() => setShowSettings(false)}>Close</button>
                </div>
            )}
        </div>
    );
};

export default ScannerView;