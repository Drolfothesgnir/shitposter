-- Fast retrieval of comments for a post, ordered by popularity
CREATE INDEX idx_comments_post_popularity ON comments(post_id, (upvotes - downvotes) DESC);

-- Alternative approach - separate index for time-based queries
CREATE INDEX idx_posts_created_at_desc ON posts(created_at DESC);
