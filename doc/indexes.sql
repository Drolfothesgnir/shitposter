CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

CREATE INDEX IF NOT EXISTS verification_emails_expires_at_idx ON verification_emails(expires_at);

CREATE INDEX IF NOT EXISTS verification_emails_user_id_secret_code_idx ON verification_emails(user_id, secret_code);

CREATE INDEX IF NOT EXISTS posts_user_id_idx ON posts(user_id);

CREATE INDEX IF NOT EXISTS posts_topics_idx ON posts USING GIN (topics);

CREATE INDEX IF NOT EXISTS comments_user_id_idx ON comments(user_id);

CREATE INDEX IF NOT EXISTS comments_post_id_idx ON comments(post_id);

-- used for faster cascade deletion of comments children
CREATE INDEX IF NOT EXISTS comments_parent_id_idx ON comments(parent_id);

CREATE UNIQUE INDEX IF NOT EXISTS post_votes_user_id_post_id ON post_votes(user_id, post_id);

-- used for faster cascade deletion of post votes
CREATE INDEX IF NOT EXISTS post_votes_post_id_idx ON post_votes(post_id);

CREATE UNIQUE INDEX IF NOT EXISTS comment_votes_user_id_comment_id ON comment_votes(user_id, comment_id);

-- used for faster cascade deletion of comment votes
CREATE INDEX IF NOT EXISTS comment_votes_comment_id_idx ON comment_votes(comment_id);

-- roots by popularity
CREATE INDEX IF NOT EXISTS comments_roots_pop
  ON comments (post_id, popularity DESC, id) WHERE parent_id IS NULL;

-- children by popularity
CREATE INDEX IF NOT EXISTS comments_children_pop
  ON comments (post_id, parent_id, popularity DESC, id);

-- added auto "popularity" column and its index to the posts
ALTER TABLE posts ADD COLUMN popularity BIGINT GENERATED ALWAYS AS (upvotes - downvotes) STORED;

-- for getting newest posts
CREATE INDEX IF NOT EXISTS idx_posts_created_at_id_desc
  ON posts (created_at DESC, id DESC);

-- for getting oldets posts
CREATE INDEX IF NOT EXISTS idx_posts_created_at_id_asc
  ON posts (created_at, id);

-- ensuring one reply per comment per user
CREATE UNIQUE INDEX IF NOT EXISTS uniq_reply_per_user_parent
  ON comments (user_id, parent_id)
  WHERE parent_id IS NOT NULL AND NOT is_deleted;