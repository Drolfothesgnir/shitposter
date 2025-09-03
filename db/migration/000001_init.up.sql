CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "username" varchar UNIQUE NOT NULL,
  "webauthn_user_handle" bytea UNIQUE NOT NULL,
  "profile_img_url" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "webauthn_credentials" (
  "id" bytea PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "public_key" bytea NOT NULL,
  "sign_count" bigint NOT NULL DEFAULT 0,
  "transports" jsonb NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_used_at" timestamptz
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "refresh_token" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "posts" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "title" varchar NOT NULL,
  "topics" jsonb,
  "body" jsonb NOT NULL,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "downvotes" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "comments" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "post_id" bigint NOT NULL,
  "parent_id" bigint,
  "depth" int NOT NULL DEFAULT 0,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "downvotes" bigint NOT NULL DEFAULT 0,
  "body" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now()),
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "post_votes" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "post_id" bigint NOT NULL,
  "vote" smallint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "comment_votes" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "comment_id" bigint NOT NULL,
  "vote" smallint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

COMMENT ON COLUMN "post_votes"."vote" IS '1 for upvote, -1 for downvote';

COMMENT ON COLUMN "comment_votes"."vote" IS '1 for upvote, -1 for downvote';

ALTER TABLE "webauthn_credentials" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "posts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "comments" ADD FOREIGN KEY ("parent_id") REFERENCES "comments" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("comment_id") REFERENCES "comments" ("id") ON DELETE CASCADE;

-- added auto "popularity" column and its index to the comments
ALTER TABLE "comments" ADD COLUMN "popularity" BIGINT GENERATED ALWAYS AS (upvotes - downvotes) STORED;
CREATE INDEX IF NOT EXISTS webauthn_credentials_user_id_idx ON webauthn_credentials(user_id);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

CREATE INDEX IF NOT EXISTS posts_user_id_idx ON posts(user_id);

CREATE INDEX IF NOT EXISTS posts_topics_idx ON posts USING GIN (topics);

CREATE INDEX IF NOT EXISTS comments_user_id_idx ON comments(user_id);

CREATE INDEX IF NOT EXISTS comments_post_id_idx ON comments(post_id);

-- used for faster cascade deletion of comments children
CREATE INDEX IF NOT EXISTS comments_parent_id_idx ON comments(parent_id);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_email_idx on users(id, email);

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
CREATE OR REPLACE FUNCTION insert_comment(
	p_user_id BIGINT,
	p_post_id BIGINT,
	p_parent_id BIGINT,
	p_body TEXT,
	p_upvotes BIGINT DEFAULT 0,
	p_downvotes BIGINT DEFAULT 0
) RETURNS comments
-- using plpgsql because I have variables and control flow
LANGUAGE plpgsql AS $$
DECLARE
	v_depth INT;
	v_post BIGINT;
  v_upvotes BIGINT := COALESCE(p_upvotes, 0);
  v_downvotes BIGINT := COALESCE(p_downvotes, 0); 
	row_out comments;
BEGIN
	IF p_parent_id IS NULL THEN
		v_depth := 0;
	ELSE
		SELECT post_id, depth INTO v_post, v_depth
		FROM comments
		WHERE id = p_parent_id
    -- disabling other transactions from deleting the parent comment
		FOR KEY SHARE;

		IF NOT FOUND THEN
			RAISE EXCEPTION 'Parent % not found', p_parent_id
			USING ERRCODE = 'foreign_key_violation';
		END IF;

		IF v_post <> p_post_id THEN
			RAISE EXCEPTION 'Parent(%) belongs to post(%) but, new comment has post(%)',
			p_parent_id, v_post, p_post_id;
		END IF;

		v_depth := v_depth + 1;
	END IF;

	INSERT INTO comments (user_id, post_id, parent_id, depth, body, upvotes, downvotes)
	VALUES (p_user_id, p_post_id, p_parent_id, v_depth, p_body, v_upvotes, v_downvotes)
	RETURNING * INTO row_out;

	RETURN row_out;
END;
$$;
-- utility for extracting comments ordered by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments
-- STABLE is used for optimization. It tells to the engine that db will not be modified, only queried
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  -- getting root comments
  roots AS (
    SELECT c.*
    FROM comments c
    WHERE c.post_id = p_post_id AND c.parent_id IS NULL
    ORDER BY c.popularity DESC, c.id
    LIMIT p_root_limit
	  OFFSET p_root_offset
  ),
  cte (
    id, user_id, post_id, parent_id, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    is_deleted, deleted_at, popularity, rnk
  ) AS (
    SELECT
      r.id, r.user_id, r.post_id, r.parent_id, r.depth,
      r.upvotes, r.downvotes, r.body, r.created_at, r.last_modified_at,
      r.is_deleted, r.deleted_at, r.popularity,
	  -- creating array of order indexes for the final sort
	  -- it gives every comment its place in ordered by popularity list
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.popularity DESC, r.id)]::BIGINT[] AS rnk
    FROM roots r

    UNION ALL

    -- getting children of the root comments
    SELECT
      ch.id, ch.user_id, ch.post_id, ch.parent_id, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.is_deleted, ch.deleted_at, ch.popularity,
      t.rnk || ch.rn AS rnk
    FROM cte t
	-- using JOIN LATERAL because the condition needs data from multiple sources
    JOIN LATERAL (
      SELECT c.*,
			-- index in ordered by popularity list, same thing as for the root comments
             ROW_NUMBER() OVER (ORDER BY c.popularity DESC, c.id) AS rn
      FROM comments c
      WHERE c.post_id = t.post_id
        AND c.parent_id = t.id
    ) ch ON TRUE
  )
  SELECT
    id, user_id, post_id, parent_id, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    is_deleted, deleted_at, popularity
  FROM cte
  -- utilising ordered index array
  ORDER BY rnk;
$$;
-- UPSERT of a comment vote
CREATE OR REPLACE FUNCTION vote_comment(
  p_user_id    bigint,
  p_comment_id bigint,
  p_vote       int
) RETURNS comments
LANGUAGE sql AS $$
WITH
-- trying to update already existed vote
upd AS (
  UPDATE comment_votes v
     SET 
      vote = p_vote,
      last_modified_at = NOW()
   WHERE v.user_id = p_user_id
     AND v.comment_id = p_comment_id
	 -- checking if two votes have different values
     AND v.vote IS DISTINCT FROM p_vote
  RETURNING
    v.comment_id,
    (CASE WHEN p_vote = 1  THEN  1 ELSE -1 END) AS up_delta,
    (CASE WHEN p_vote = -1 THEN  1 ELSE -1 END) AS down_delta
),

ins AS (
  INSERT INTO comment_votes(user_id, comment_id, vote)
  VALUES (p_user_id, p_comment_id, p_vote)
  -- if another transaction is already trying to create new vote do nothing
  ON CONFLICT (user_id, comment_id) DO NOTHING
  RETURNING
    comment_id,
    (p_vote = 1)::int  AS up_delta,
    (p_vote = -1)::int AS down_delta
),
-- extracting deltas from whatever operation succeeded
delta AS (
  SELECT * FROM upd
  UNION ALL
  SELECT * FROM ins
),
-- applying deltas to the comments counters
bump AS (
  UPDATE comments c
     SET upvotes   = c.upvotes   + COALESCE(d.up_delta, 0),
         downvotes = c.downvotes + COALESCE(d.down_delta, 0)
    FROM delta d
   WHERE c.id = d.comment_id
  RETURNING c.*
)
-- returning updated comment
SELECT *
FROM bump 
UNION ALL
SELECT c.*
FROM comments c
-- check in comments only if bump didn't return anything
WHERE c.id = p_comment_id
	AND NOT EXISTS (SELECT 1 FROM bump);
$$;
-- UPSERT of a post vote
CREATE OR REPLACE FUNCTION vote_post(
  p_user_id    bigint,
  p_post_id bigint,
  p_vote       int
) RETURNS posts
LANGUAGE sql AS $$
WITH
-- trying to update already existed vote
upd AS (
  UPDATE post_votes v
     SET 
      vote = p_vote,
      last_modified_at = NOW()
   WHERE v.user_id = p_user_id
     AND v.post_id = p_post_id
	 -- checking if two votes have different values
     AND v.vote IS DISTINCT FROM p_vote
  RETURNING
    v.post_id,
    (CASE WHEN p_vote = 1  THEN  1 ELSE -1 END) AS up_delta,
    (CASE WHEN p_vote = -1 THEN  1 ELSE -1 END) AS down_delta
),

ins AS (
  INSERT INTO post_votes(user_id, post_id, vote)
  VALUES (p_user_id, p_post_id, p_vote)
  -- if another transaction is already trying to create new vote do nothing
  ON CONFLICT (user_id, post_id) DO NOTHING
  RETURNING
    post_id,
    (p_vote = 1)::int  AS up_delta,
    (p_vote = -1)::int AS down_delta
),
-- extracting deltas from whatever operation succeeded
delta AS (
  SELECT * FROM upd
  UNION ALL
  SELECT * FROM ins
),
-- applying deltas to the posts counters
bump AS (
  UPDATE posts p
     SET upvotes   = p.upvotes   + COALESCE(d.up_delta, 0),
         downvotes = p.downvotes + COALESCE(d.down_delta, 0)
    FROM delta d
   WHERE p.id = d.post_id
  RETURNING p.*
)
-- returning updated post
SELECT *
FROM bump 
UNION ALL
SELECT p.*
FROM posts p
-- check in posts only if bump didn't return anything
WHERE p.id = p_post_id
	AND NOT EXISTS (SELECT 1 FROM bump);
$$;CREATE OR REPLACE FUNCTION delete_comment_vote(
	p_comment_id BIGINT,
	p_user_id BIGINT
) RETURNS void
LANGUAGE sql AS $$
WITH del AS (
	DELETE FROM comment_votes
	WHERE user_id = p_user_id AND comment_id = p_comment_id
	RETURNING vote
)
UPDATE comments c
SET
	upvotes = c.upvotes + (CASE WHEN d.vote = 1 THEN -1 ELSE 0 END),
	downvotes = c.downvotes + (CASE WHEN d.vote = -1 THEN -1 ELSE 0 END)
FROM del d
WHERE c.id = p_comment_id;
$$;CREATE OR REPLACE FUNCTION delete_post_vote(
	p_post_id BIGINT,
	p_user_id BIGINT
) RETURNS void
LANGUAGE sql AS $$
WITH del AS (
	DELETE FROM post_votes
	WHERE user_id = p_user_id AND post_id = p_post_id
	RETURNING vote
)
UPDATE posts p
SET
	upvotes = p.upvotes + (CASE WHEN d.vote = 1 THEN -1 ELSE 0 END),
	downvotes = p.downvotes + (CASE WHEN d.vote = -1 THEN -1 ELSE 0 END)
FROM del d
WHERE p.id = p_post_id;
$$;
