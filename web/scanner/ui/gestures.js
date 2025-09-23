/**
 * Gesture Recognition Module
 * Handles touch gestures for camera controls (pinch zoom, tap-to-focus, double-tap)
 */

class GestureManager {
    constructor(element, options = {}) {
        this.element = element;
        this.enabled = true;

        this.config = {
            // Pinch zoom
            enablePinchZoom: true,
            minZoom: 1.0,
            maxZoom: 8.0,
            zoomSpeed: 0.01,

            // Tap to focus
            enableTapToFocus: true,
            tapTimeout: 300,
            tapRadius: 20,

            // Double tap zoom
            enableDoubleTap: true,
            doubleTapTimeout: 300,
            doubleTapZoomLevel: 2.0,

            // Visual feedback
            showFocusIndicator: true,
            focusIndicatorDuration: 1000,

            ...options
        };

        this.state = {
            // Pinch state
            isPinching: false,
            initialDistance: 0,
            initialZoom: 1.0,
            currentZoom: 1.0,

            // Tap state
            lastTapTime: 0,
            lastTapPosition: { x: 0, y: 0 },
            tapCount: 0,

            // Touch tracking
            touches: new Map(),
            isTouch: false
        };

        this.eventListeners = {
            pinchZoom: [],
            tapToFocus: [],
            doubleTap: [],
            gestureStart: [],
            gestureEnd: []
        };

        this.focusIndicator = null;
        this.setupEventListeners();
    }

    /**
     * Set up event listeners
     */
    setupEventListeners() {
        // Touch events
        this.element.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
        this.element.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
        this.element.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: false });
        this.element.addEventListener('touchcancel', this.handleTouchCancel.bind(this), { passive: false });

        // Mouse events (for desktop testing)
        this.element.addEventListener('click', this.handleClick.bind(this));
        this.element.addEventListener('wheel', this.handleWheel.bind(this), { passive: false });

        // Pointer events (if supported)
        if (window.PointerEvent) {
            this.element.addEventListener('pointerdown', this.handlePointerDown.bind(this));
            this.element.addEventListener('pointermove', this.handlePointerMove.bind(this));
            this.element.addEventListener('pointerup', this.handlePointerUp.bind(this));
            this.element.addEventListener('pointercancel', this.handlePointerCancel.bind(this));
        }

        // Prevent default behaviors
        this.element.addEventListener('contextmenu', (e) => e.preventDefault());
        this.element.addEventListener('selectstart', (e) => e.preventDefault());
    }

    /**
     * Touch start handler
     */
    handleTouchStart(event) {
        if (!this.enabled) return;

        this.state.isTouch = true;

        // Update touch tracking
        for (const touch of event.changedTouches) {
            this.state.touches.set(touch.identifier, {
                x: touch.clientX,
                y: touch.clientY,
                startTime: Date.now()
            });
        }

        const touchCount = this.state.touches.size;

        if (touchCount === 1) {
            // Single touch - potential tap
            this.handleSingleTouchStart(event.changedTouches[0]);
        } else if (touchCount === 2 && this.config.enablePinchZoom) {
            // Two touches - start pinch gesture
            this.handlePinchStart(event);
            event.preventDefault();
        }

        this.emit('gestureStart', { type: 'touch', touchCount });
    }

    /**
     * Touch move handler
     */
    handleTouchMove(event) {
        if (!this.enabled || !this.state.isTouch) return;

        // Update touch positions
        for (const touch of event.changedTouches) {
            if (this.state.touches.has(touch.identifier)) {
                this.state.touches.set(touch.identifier, {
                    ...this.state.touches.get(touch.identifier),
                    x: touch.clientX,
                    y: touch.clientY
                });
            }
        }

        if (this.state.touches.size === 2 && this.state.isPinching) {
            this.handlePinchMove(event);
            event.preventDefault();
        }
    }

    /**
     * Touch end handler
     */
    handleTouchEnd(event) {
        if (!this.enabled) return;

        // Process ended touches
        for (const touch of event.changedTouches) {
            if (this.state.touches.has(touch.identifier)) {
                const touchData = this.state.touches.get(touch.identifier);
                const duration = Date.now() - touchData.startTime;

                // Check for tap gesture
                if (duration < this.config.tapTimeout && !this.state.isPinching) {
                    this.handleTap(touch, duration);
                }

                this.state.touches.delete(touch.identifier);
            }
        }

        // End pinch if no more touches
        if (this.state.touches.size < 2 && this.state.isPinching) {
            this.handlePinchEnd();
        }

        // Reset touch state if no touches remain
        if (this.state.touches.size === 0) {
            this.state.isTouch = false;
            this.emit('gestureEnd', { type: 'touch' });
        }
    }

    /**
     * Touch cancel handler
     */
    handleTouchCancel(event) {
        this.handleTouchEnd(event);
    }

    /**
     * Single touch start
     */
    handleSingleTouchStart(touch) {
        // Store touch for potential tap
        this.state.lastTouchStart = {
            x: touch.clientX,
            y: touch.clientY,
            time: Date.now()
        };
    }

    /**
     * Pinch start
     */
    handlePinchStart(event) {
        const touches = Array.from(this.state.touches.values());
        if (touches.length < 2) return;

        this.state.isPinching = true;
        this.state.initialDistance = this.getDistance(touches[0], touches[1]);
        this.state.initialZoom = this.state.currentZoom;

        console.log('[GestureManager] Pinch started, initial distance:', this.state.initialDistance);
    }

    /**
     * Pinch move
     */
    handlePinchMove(event) {
        const touches = Array.from(this.state.touches.values());
        if (touches.length < 2 || !this.state.isPinching) return;

        const currentDistance = this.getDistance(touches[0], touches[1]);
        const scale = currentDistance / this.state.initialDistance;

        let newZoom = this.state.initialZoom * scale;
        newZoom = Math.max(this.config.minZoom, Math.min(this.config.maxZoom, newZoom));

        if (newZoom !== this.state.currentZoom) {
            this.state.currentZoom = newZoom;
            this.emit('pinchZoom', {
                zoom: newZoom,
                scale: scale,
                center: this.getCenter(touches[0], touches[1])
            });
        }
    }

    /**
     * Pinch end
     */
    handlePinchEnd() {
        console.log('[GestureManager] Pinch ended, final zoom:', this.state.currentZoom);
        this.state.isPinching = false;
        this.state.initialDistance = 0;
    }

    /**
     * Handle tap gesture
     */
    handleTap(touch, duration) {
        if (!this.config.enableTapToFocus && !this.config.enableDoubleTap) {
            return;
        }

        const position = this.getRelativePosition(touch.clientX, touch.clientY);
        const now = Date.now();

        // Check for double tap
        if (this.config.enableDoubleTap) {
            const timeSinceLastTap = now - this.state.lastTapTime;
            const distance = this.getDistance(position, this.state.lastTapPosition);

            if (timeSinceLastTap < this.config.doubleTapTimeout && distance < this.config.tapRadius) {
                this.handleDoubleTap(position);
                this.state.tapCount = 0;
                return;
            }
        }

        // Single tap (with delay to check for double tap)
        this.state.lastTapTime = now;
        this.state.lastTapPosition = position;
        this.state.tapCount++;

        if (this.config.enableTapToFocus) {
            setTimeout(() => {
                if (this.state.tapCount === 1) {
                    this.handleSingleTap(position);
                }
                this.state.tapCount = 0;
            }, this.config.doubleTapTimeout);
        }
    }

    /**
     * Handle single tap (tap-to-focus)
     */
    handleSingleTap(position) {
        console.log('[GestureManager] Single tap at:', position);

        this.emit('tapToFocus', {
            x: position.x,
            y: position.y,
            relative: {
                x: position.x / this.element.clientWidth,
                y: position.y / this.element.clientHeight
            }
        });

        if (this.config.showFocusIndicator) {
            this.showFocusIndicator(position.x, position.y);
        }
    }

    /**
     * Handle double tap (zoom toggle)
     */
    handleDoubleTap(position) {
        console.log('[GestureManager] Double tap at:', position);

        const targetZoom = this.state.currentZoom > 1.5 ? 1.0 : this.config.doubleTapZoomLevel;

        this.emit('doubleTap', {
            x: position.x,
            y: position.y,
            targetZoom: targetZoom,
            currentZoom: this.state.currentZoom
        });

        // Animate zoom change
        this.animateZoomTo(targetZoom);
    }

    /**
     * Mouse click handler (desktop)
     */
    handleClick(event) {
        if (this.state.isTouch) return; // Ignore if touch is active

        const position = this.getRelativePosition(event.clientX, event.clientY);
        this.handleSingleTap(position);
    }

    /**
     * Mouse wheel handler (desktop zoom)
     */
    handleWheel(event) {
        if (!this.config.enablePinchZoom) return;

        event.preventDefault();

        const delta = event.deltaY > 0 ? -0.1 : 0.1;
        let newZoom = this.state.currentZoom + delta;
        newZoom = Math.max(this.config.minZoom, Math.min(this.config.maxZoom, newZoom));

        if (newZoom !== this.state.currentZoom) {
            this.state.currentZoom = newZoom;
            this.emit('pinchZoom', {
                zoom: newZoom,
                scale: newZoom / this.state.initialZoom,
                center: this.getRelativePosition(event.clientX, event.clientY)
            });
        }
    }

    /**
     * Pointer event handlers (for hybrid devices)
     */
    handlePointerDown(event) {
        // Let touch events handle touch input
        if (event.pointerType === 'touch') return;

        // Handle mouse/pen input
        this.state.pointerStart = {
            x: event.clientX,
            y: event.clientY,
            time: Date.now()
        };
    }

    handlePointerMove(event) {
        // Handle pointer move if needed
    }

    handlePointerUp(event) {
        if (event.pointerType === 'touch') return;

        if (this.state.pointerStart) {
            const duration = Date.now() - this.state.pointerStart.time;
            const distance = this.getDistance(
                { x: event.clientX, y: event.clientY },
                this.state.pointerStart
            );

            if (duration < this.config.tapTimeout && distance < this.config.tapRadius) {
                const position = this.getRelativePosition(event.clientX, event.clientY);
                this.handleSingleTap(position);
            }
        }

        this.state.pointerStart = null;
    }

    handlePointerCancel(event) {
        this.state.pointerStart = null;
    }

    /**
     * Show focus indicator
     */
    showFocusIndicator(x, y) {
        // Remove existing indicator
        this.hideFocusIndicator();

        // Create focus indicator
        this.focusIndicator = document.createElement('div');
        this.focusIndicator.className = 'scanner-focus-indicator';
        this.focusIndicator.style.cssText = `
            position: absolute;
            left: ${x - 25}px;
            top: ${y - 25}px;
            width: 50px;
            height: 50px;
            border: 2px solid #00ff00;
            border-radius: 50%;
            pointer-events: none;
            animation: focusPulse 0.6s ease-out;
            z-index: 1000;
        `;

        // Add CSS animation if not exists
        if (!document.getElementById('scanner-focus-styles')) {
            const style = document.createElement('style');
            style.id = 'scanner-focus-styles';
            style.textContent = `
                @keyframes focusPulse {
                    0% { transform: scale(1.5); opacity: 0; }
                    50% { transform: scale(1); opacity: 1; }
                    100% { transform: scale(1); opacity: 0.7; }
                }
            `;
            document.head.appendChild(style);
        }

        this.element.appendChild(this.focusIndicator);

        // Auto-hide after duration
        setTimeout(() => {
            this.hideFocusIndicator();
        }, this.config.focusIndicatorDuration);
    }

    /**
     * Hide focus indicator
     */
    hideFocusIndicator() {
        if (this.focusIndicator) {
            this.focusIndicator.remove();
            this.focusIndicator = null;
        }
    }

    /**
     * Animate zoom to target level
     */
    animateZoomTo(targetZoom, duration = 300) {
        const startZoom = this.state.currentZoom;
        const startTime = Date.now();

        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);

            // Ease-out animation
            const easeProgress = 1 - Math.pow(1 - progress, 3);
            const currentZoom = startZoom + (targetZoom - startZoom) * easeProgress;

            this.state.currentZoom = currentZoom;
            this.emit('pinchZoom', {
                zoom: currentZoom,
                scale: currentZoom / this.state.initialZoom,
                center: { x: this.element.clientWidth / 2, y: this.element.clientHeight / 2 },
                animated: true
            });

            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    }

    /**
     * Utility functions
     */
    getDistance(point1, point2) {
        const dx = point2.x - point1.x;
        const dy = point2.y - point1.y;
        return Math.sqrt(dx * dx + dy * dy);
    }

    getCenter(point1, point2) {
        return {
            x: (point1.x + point2.x) / 2,
            y: (point1.y + point2.y) / 2
        };
    }

    getRelativePosition(clientX, clientY) {
        const rect = this.element.getBoundingClientRect();
        return {
            x: clientX - rect.left,
            y: clientY - rect.top
        };
    }

    /**
     * Set zoom level programmatically
     */
    setZoom(zoom) {
        this.state.currentZoom = Math.max(this.config.minZoom, Math.min(this.config.maxZoom, zoom));
        return this.state.currentZoom;
    }

    /**
     * Get current zoom level
     */
    getZoom() {
        return this.state.currentZoom;
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
                console.error('[GestureManager] Event callback error:', error);
            }
        });
    }

    /**
     * Enable/disable gestures
     */
    setEnabled(enabled) {
        this.enabled = enabled;
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
        this.hideFocusIndicator();
        this.eventListeners = {
            pinchZoom: [],
            tapToFocus: [],
            doubleTap: [],
            gestureStart: [],
            gestureEnd: []
        };
    }
}

// Make available globally and as module
if (typeof window !== 'undefined') {
    window.GestureManager = GestureManager;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = GestureManager;
}