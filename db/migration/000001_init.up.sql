CREATE EXTENSION IF NOT EXISTS ltree;
-- SQL dump generated using DBML (dbml.dbdiagram.io)
-- Database: PostgreSQL
-- Generated at: 2025-08-19T20:14:14.701Z

CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "username" varchar UNIQUE NOT NULL,
  "profile_img_url" varchar NOT NULL,
  "hashed_password" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "is_email_verified" bool NOT NULL DEFAULT false,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "refresh_token" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "verification_emails" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "email" varchar NOT NULL,
  "secret_code" varchar NOT NULL,
  "is_used" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "expires_at" timestamptz NOT NULL DEFAULT (now() + interval '15 minutes')
);

CREATE TABLE "posts" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
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
  "user_id" bigserial NOT NULL,
  "post_id" bigserial NOT NULL,
  "path" ltree NOT NULL,
  "depth" int NOT NULL DEFAULT 0,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "downvotes" bigint NOT NULL DEFAULT 0,
  "body" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "post_votes" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "post_id" bigserial NOT NULL,
  "vote" int8 NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "comment_votes" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "comment_id" bigserial NOT NULL,
  "vote" int8 NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_modified_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "sessions" ("user_id");

CREATE INDEX ON "sessions" ("expires_at");

CREATE INDEX ON "verification_emails" ("expires_at");

CREATE INDEX ON "verification_emails" ("user_id", "secret_code");

CREATE INDEX ON "posts" ("user_id");

CREATE INDEX ON "posts" USING GIN ("topics");

CREATE INDEX ON "comments" ("user_id");

CREATE INDEX ON "comments" ("post_id");

CREATE INDEX ON "comments" USING GIST ("path");

CREATE INDEX ON "comments" ("path", "depth");

CREATE UNIQUE INDEX ON "post_votes" ("user_id", "post_id");

CREATE UNIQUE INDEX ON "comment_votes" ("user_id", "comment_id");

COMMENT ON COLUMN "post_votes"."vote" IS '1 for upvote, -1 for downvote';

COMMENT ON COLUMN "comment_votes"."vote" IS '1 for upvote, -1 for downvote';

ALTER TABLE "sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "verification_emails" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "posts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("comment_id") REFERENCES "comments" ("id") ON DELETE CASCADE;
-- Fast retrieval of comments for a post, ordered by popularity
CREATE INDEX idx_comments_post_popularity ON comments(post_id, (upvotes - downvotes) DESC);

-- Alternative approach - separate index for time-based queries
CREATE INDEX idx_posts_created_at_desc ON posts(created_at DESC);

-- Composite index for the anchor query (root comments)
CREATE INDEX idx_comments_post_depth_popularity 
ON comments (post_id, depth, (upvotes - downvotes) DESC);
CREATE OR REPLACE FUNCTION insert_comment(
	p_user_id BIGINT,
	p_post_id BIGINT,
	p_parent_path LTREE,
	p_body TEXT,
	p_upvotes BIGINT DEFAULT 0,
	p_downvotes BIGINT DEFAULT 0
) RETURNS comments AS $$
DECLARE
	new_id BIGINT;
	new_path LTREE;
	new_depth INT;
	result comments;
BEGIN
	new_id := NEXTVAL('comments_id_seq');
	IF p_parent_path IS NULL THEN
		new_path := new_id::TEXT::LTREE;
		new_depth := 0;
	ELSE
		new_path := (p_parent_path::TEXT || '.' || new_id::TEXT)::LTREE;
    	new_depth := NLEVEL(p_parent_path);
	END IF;

	INSERT INTO comments (id, user_id, post_id, path, depth, body, upvotes, downvotes)
	VALUES (new_id, p_user_id, p_post_id, new_path, new_depth, p_body, p_upvotes, p_downvotes)
	RETURNING * INTO result;

	RETURN result;
END;

$$ LANGUAGE plpgsql;


-- utility for ordering comments recursively depth-first by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
	p_post_id BIGINT,
	p_root_comments_limit BIGINT
) RETURNS SETOF comments AS $$
	BEGIN
		RETURN QUERY
		WITH RECURSIVE cte AS (
			SELECT c.*, 
				-- root comment order index used as LIMIT
				ROW_NUMBER() OVER(ORDER BY (c.upvotes - c.downvotes) DESC) AS rn,
				-- rank used for the end sorting 
				(ROW_NUMBER() OVER(ORDER BY (c.upvotes - c.downvotes) DESC))::text::ltree AS rank
			FROM comments c
			WHERE c.depth = 0 AND c.post_id = p_post_id
		
			UNION ALL
		
			SELECT c.*, 
				t.rn,
				-- concatenate rank to the parent index to get rank
				t.rank || (ROW_NUMBER() OVER(ORDER BY (c.upvotes - c.downvotes) DESC))::text AS rank
				
			FROM comments c, cte t
			-- checks if comment is a descendant of one of the previously found comments
			-- and if there is not too much root comments found
			WHERE c.path <@ t.path AND c.depth = t.depth + 1 AND t.rn <= p_root_comments_limit
		)
		SELECT 
			c.id, 
			c.user_id, 
			c.post_id, 
			c.path, 
			c.depth, 
			c.upvotes, 
			c.downvotes,
			c.body, 
			c.created_at, 
			c.last_modified_at
		FROM cte c
		WHERE rn <= p_root_comments_limit
		ORDER BY rank;
	END;

$$ LANGUAGE plpgsql;