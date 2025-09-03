-- ================================================================
-- MIGRATION 014: ADD MISSING WEBAUTHN SESSION TABLE
-- Creates webauthn_sessions table if it doesn't exist properly
-- ================================================================

-- Drop and recreate the table to ensure proper structure
DROP TABLE IF EXISTS webauthn_sessions;

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