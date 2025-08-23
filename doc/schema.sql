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
  "parent_id" bigserial,
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

CREATE UNIQUE INDEX ON "post_votes" ("user_id", "post_id");

CREATE UNIQUE INDEX ON "comment_votes" ("user_id", "comment_id");

COMMENT ON COLUMN "post_votes"."vote" IS '1 for upvote, -1 for downvote';

COMMENT ON COLUMN "comment_votes"."vote" IS '1 for upvote, -1 for downvote';

ALTER TABLE "sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "verification_emails" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "posts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "comments" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "comments" ADD FOREIGN KEY ("parent_id") REFERENCES "comments" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "post_votes" ADD FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE;

ALTER TABLE "comment_votes" ADD FOREIGN KEY ("comment_id") REFERENCES "comments" ("id") ON DELETE CASCADE;

-- added auto "popularity" column and its index
ALTER TABLE comments ADD COLUMN popularity BIGINT GENERATED ALWAYS AS (upvotes - downvotes) STORED;

-- Alternative approach - separate index for time-based queries
CREATE INDEX idx_posts_created_at_desc ON posts(created_at DESC);

CREATE OR REPLACE FUNCTION insert_comment(
	p_user_id BIGINT,
	p_post_id BIGINT,
	p_parent_id BIGINT,
	p_body TEXT,
	p_upvotes BIGINT DEFAULT 0,
	p_downvotes BIGINT DEFAULT 0
) RETURNS comments
LANGUAGE plpgsql AS $$
DECLARE
	v_depth INT;
	v_post BIGINT;
	row_out comments;
BEGIN
	IF p_parent_id IS NULL THEN
		v_depth := 0;
	ELSE
		SELECT post_id, depth INTO v_post, v_depth
		FROM comments
		WHERE id = p_parent_id
		FOR SHARE;

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
	VALUES (p_user_id, p_post_id, p_parent_id, v_depth, p_body, p_upvotes, p_downvotes)
	RETURNING * INTO row_out;

	RETURN row_out;
END;
$$;


-- utility for extracting comments ordered by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
  p_post_id BIGINT,
  p_root_limit INT
) RETURNS SETOF comments
-- STABLE is used for optimization. It tells to the engine that db will not be modified, only queried
LANGUAGE plpgsql STABLE AS $$
BEGIN
  RETURN QUERY
  WITH RECURSIVE
  -- getting roor comments
  roots AS (
    SELECT c.*
    FROM comments c
    WHERE c.post_id = p_post_id AND c.parent_id IS NULL
    ORDER BY c.popularity DESC, c.id
    LIMIT p_root_limit
  ),
  cte (
    id, user_id, post_id, path, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    parent_id, popularity, rnk
  ) AS (
    SELECT
      r.id, r.user_id, r.post_id, r.path, r.depth,
      r.upvotes, r.downvotes, r.body, r.created_at, r.last_modified_at,
      r.parent_id, r.popularity,
	  -- creating array of order indexes for the final sort
	  -- it gives every comment its place in ordered by popularity list
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.popularity DESC, r.id)] AS rnk
    FROM roots r

    UNION ALL

    -- getting children of the root comments
    SELECT
      ch.id, ch.user_id, ch.post_id, ch.path, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.parent_id, ch.popularity,
      t.rnk || ch.rn AS rnk
    FROM cte t
	-- using JOIN LATERAL because the condition needs data from multiple sources
	-- and I didn't used FROM comments c, cte t because... I don't know
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
    id, user_id, post_id, path, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    popularity, parent_id
  FROM cte
  -- utilising ordered index array
  ORDER BY rnk;
END;
$$;