-- Drop the procedure if it exists to ensure a clean run
DROP PROCEDURE IF EXISTS fix_sessions_and_users;

DELIMITER $$

-- Create a procedure to encapsulate the migration logic
CREATE PROCEDURE fix_sessions_and_users()
BEGIN
    -- Check if a primary key already exists on the sessions table.
    SET @pk_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'sessions' AND CONSTRAINT_TYPE = 'PRIMARY KEY');

    -- Conditionally add the primary key to the sessions table on session_id.
    SET @sql_pk = IF(@pk_exists = 0, 'ALTER TABLE `sessions` ADD PRIMARY KEY (`session_id`)', 'SELECT "Primary key already exists on sessions table, skipping."');
    PREPARE stmt_pk FROM @sql_pk;
    EXECUTE stmt_pk;
    DEALLOCATE PREPARE stmt_pk;

    -- The user_id in the sessions table is not unique, so a foreign key from users to sessions is not possible.
    -- This appears to be a logic error in a previous migration.
    -- We will attempt to drop this constraint if it exists.
    SET @fk_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND CONSTRAINT_NAME = 'fk_sessions_user');

    -- Conditionally drop the foreign key
    SET @sql_fk = IF(@fk_exists > 0, 'ALTER TABLE `users` DROP FOREIGN KEY `fk_sessions_user`', 'SELECT "Foreign key fk_sessions_user does not exist, skipping."');
    PREPARE stmt_fk FROM @sql_fk;
    EXECUTE stmt_fk;
    DEALLOCATE PREPARE stmt_fk;
END$$

DELIMITER ;

-- Execute the stored procedure to apply the fixes
CALL fix_sessions_and_users();

-- Clean up by dropping the procedure
DROP PROCEDURE fix_sessions_and_users;
