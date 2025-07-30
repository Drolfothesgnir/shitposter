-- Down migration script to undo all database changes
-- Run this to completely remove the shitposter database schema

-- Drop functions first
DROP FUNCTION IF EXISTS insert_comment(bigint, bigint, ltree, text);

-- Drop custom indexes (expression indexes and named indexes)
DROP INDEX IF EXISTS idx_comments_post_popularity;
DROP INDEX IF EXISTS idx_comments_parent_id; 
DROP INDEX IF EXISTS idx_posts_created_at_desc;

-- Drop all tables (in reverse dependency order)
DROP TABLE IF EXISTS comment_votes CASCADE;
DROP TABLE IF EXISTS post_votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS verification_emails CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop sequences (if they weren't automatically dropped with tables)
DROP SEQUENCE IF EXISTS comments_id_seq;
DROP SEQUENCE IF EXISTS posts_id_seq;
DROP SEQUENCE IF EXISTS users_id_seq;
DROP SEQUENCE IF EXISTS verification_emails_id_seq;

-- Drop extensions (be careful - other databases might use these)
-- Only uncomment if you're sure no other schemas need these extensions
DROP EXTENSION IF EXISTS ltree;
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- Optional: Drop the entire schema if you created a custom one
-- DROP SCHEMA IF EXISTS shitposter CASCADE;