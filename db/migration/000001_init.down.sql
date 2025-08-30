-- Migration Down Script
-- This script reverses all changes from the up migration

-- Drop all functions
DROP FUNCTION IF EXISTS delete_post_vote(BIGINT, BIGINT);
DROP FUNCTION IF EXISTS delete_comment_vote(BIGINT, BIGINT);
DROP FUNCTION IF EXISTS vote_post(BIGINT, BIGINT, INT);
DROP FUNCTION IF EXISTS vote_comment(BIGINT, BIGINT, INT);
DROP FUNCTION IF EXISTS get_comments_by_popularity(BIGINT, INT, INT);
DROP FUNCTION IF EXISTS insert_comment(BIGINT, BIGINT, BIGINT, TEXT, BIGINT, BIGINT);

-- Drop all indexes (in reverse order of creation)
DROP INDEX IF EXISTS idx_posts_created_at_id_asc;
DROP INDEX IF EXISTS idx_posts_created_at_id_desc;
DROP INDEX IF EXISTS comments_children_pop;
DROP INDEX IF EXISTS comments_roots_pop;
DROP INDEX IF EXISTS comment_votes_comment_id_idx;
DROP INDEX IF EXISTS comment_votes_user_id_comment_id;
DROP INDEX IF EXISTS post_votes_post_id_idx;
DROP INDEX IF EXISTS post_votes_user_id_post_id;
DROP INDEX IF EXISTS comments_parent_id_idx;
DROP INDEX IF EXISTS comments_post_id_idx;
DROP INDEX IF EXISTS comments_user_id_idx;
DROP INDEX IF EXISTS posts_topics_idx;
DROP INDEX IF EXISTS posts_user_id_idx;
DROP INDEX IF EXISTS verification_emails_user_id_secret_code_idx;
DROP INDEX IF EXISTS verification_emails_expires_at_idx;
DROP INDEX IF EXISTS sessions_expires_at_idx;
DROP INDEX IF EXISTS sessions_user_id_idx;

-- Remove generated columns
ALTER TABLE posts DROP COLUMN IF EXISTS popularity;
ALTER TABLE comments DROP COLUMN IF EXISTS popularity;

-- Drop all foreign key constraints
-- Note: In PostgreSQL, foreign keys are typically named automatically
-- We'll use CASCADE to handle dependencies
ALTER TABLE comment_votes DROP CONSTRAINT IF EXISTS comment_votes_comment_id_fkey;
ALTER TABLE comment_votes DROP CONSTRAINT IF EXISTS comment_votes_user_id_fkey;
ALTER TABLE post_votes DROP CONSTRAINT IF EXISTS post_votes_post_id_fkey;
ALTER TABLE post_votes DROP CONSTRAINT IF EXISTS post_votes_user_id_fkey;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS comments_parent_id_fkey;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS comments_post_id_fkey;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS comments_user_id_fkey;
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_user_id_fkey;
ALTER TABLE verification_emails DROP CONSTRAINT IF EXISTS verification_emails_user_id_fkey;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_user_id_fkey;

-- Drop all tables (in reverse order of dependencies)
DROP TABLE IF EXISTS comment_votes CASCADE;
DROP TABLE IF EXISTS post_votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS verification_emails CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS users CASCADE;