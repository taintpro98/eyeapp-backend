DROP TRIGGER IF EXISTS update_user_identities_updated_at ON user_identities;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS users;
