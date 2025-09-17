-- Create job_attachments table for file attachments to jobs
CREATE TABLE job_attachments (
    attachment_id INT AUTO_INCREMENT PRIMARY KEY,
    job_id INT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    uploaded_by INT DEFAULT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (job_id) REFERENCES jobs(jobID) ON DELETE CASCADE,
    FOREIGN KEY (uploaded_by) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_job_attachments_job_id (job_id),
    INDEX idx_job_attachments_uploaded_at (uploaded_at)
);