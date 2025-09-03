-- Fix Foreign Key Constraint for User Preferences
-- The constraint was backwards - users should not reference user_preferences
-- Instead, user_preferences should reference users

-- First, drop the incorrect foreign key constraint
ALTER TABLE `users` DROP FOREIGN KEY `fk_user_preferences_user`;

-- Then, add the correct foreign key constraint on the user_preferences table
-- This ensures user_preferences.user_id references users.userID (correct direction)
ALTER TABLE `user_preferences` 
  ADD CONSTRAINT `fk_user_preferences_user` 
  FOREIGN KEY (`user_id`) REFERENCES `users` (`userID`) 
  ON DELETE CASCADE ON UPDATE CASCADE;