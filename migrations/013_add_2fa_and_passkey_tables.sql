-- ================================================================
-- MIGRATION 013: 2FA AND PASSKEY AUTHENTICATION TABLES
-- Creates the missing tables for two-factor authentication and passkeys
-- ================================================================

-- Start transaction for atomic migration
START TRANSACTION;

-- ================================================================
-- 1. TWO-FACTOR AUTHENTICATION TABLE
-- ================================================================

CREATE TABLE user_2fa (
    two_fa_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    secret VARCHAR(100) NOT NULL,
    qr_code_url TEXT,
    backup_codes JSON,
    is_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE,
    UNIQUE KEY unique_user_2fa (user_id),
    INDEX idx_user_enabled (user_id, is_enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 2. USER PASSKEYS TABLE
-- ================================================================

CREATE TABLE user_passkeys (
    passkey_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    credential_id VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    sign_count INT DEFAULT 0,
    aaguid VARCHAR(36),
    transport_methods JSON,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE,
    INDEX idx_user_active (user_id, is_active),
    INDEX idx_credential (credential_id),
    INDEX idx_last_used (last_used_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 3. AUTHENTICATION ATTEMPTS TABLE
-- ================================================================

CREATE TABLE authentication_attempts (
    attempt_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED,
    method ENUM('password', '2fa', 'passkey', 'backup_code') NOT NULL,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    failure_reason VARCHAR(255),
    session_id VARCHAR(191),
    FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_user_time (user_id, attempted_at),
    INDEX idx_method_success (method, success),
    INDEX idx_ip_time (ip_address, attempted_at),
    INDEX idx_attempted_at (attempted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 4. USER PREFERENCES TABLE (Enhanced)
-- ================================================================

CREATE TABLE user_preferences (
    preference_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    language VARCHAR(5) DEFAULT 'de',
    theme VARCHAR(10) DEFAULT 'dark',
    time_zone VARCHAR(50) DEFAULT 'Europe/Berlin',
    date_format VARCHAR(20) DEFAULT 'DD.MM.YYYY',
    time_format VARCHAR(5) DEFAULT '24h',
    email_notifications BOOLEAN DEFAULT TRUE,
    system_notifications BOOLEAN DEFAULT TRUE,
    job_status_notifications BOOLEAN DEFAULT TRUE,
    device_alert_notifications BOOLEAN DEFAULT TRUE,
    items_per_page INT DEFAULT 25,
    default_view VARCHAR(20) DEFAULT 'list',
    show_advanced_options BOOLEAN DEFAULT FALSE,
    auto_save_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE,
    UNIQUE KEY unique_user_preferences (user_id),
    INDEX idx_user_prefs (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 5. WEBAUTHN SESSION TABLE
-- ================================================================

CREATE TABLE webauthn_sessions (
    session_id VARCHAR(191) PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    challenge VARCHAR(255) NOT NULL,
    session_type VARCHAR(50) NOT NULL,
    session_data TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_session (user_id, session_type),
    INDEX idx_expires (expires_at),
    INDEX idx_session_type (session_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 6. SESSION MANAGEMENT TABLE (Enhanced)
-- ================================================================

CREATE TABLE user_sessions (
    session_id VARCHAR(191) PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    device_info JSON,
    FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE,
    INDEX idx_user_active (user_id, is_active),
    INDEX idx_expires (expires_at),
    INDEX idx_last_active (last_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Commit the migration
COMMIT;

-- ================================================================
-- MIGRATION COMPLETE
-- ================================================================
-- This migration adds:
-- ✅ Complete 2FA support with backup codes
-- ✅ WebAuthn passkey authentication
-- ✅ Authentication attempt logging
-- ✅ Enhanced user preferences
-- ✅ Session management
-- ================================================================