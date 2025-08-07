CREATE EXTENSION IF NOT EXISTS ltree;
-- SQL dump generated using DBML (dbml.dbdiagram.io)
-- Database: PostgreSQL
-- Generated at: 2025-08-06T13:07:16.534Z

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
create or replace function insert_comment(
	p_user_id bigint,
	p_post_id bigint,
	p_parent_path ltree,
	p_body text,
	p_upvotes bigint default 0,
	p_downvotes bigint default 0
) returns comments as $$
declare
	new_id bigint;
	new_path ltree;
	new_depth int;
	result comments;
begin
	new_id := nextval('comments_id_seq');
	if p_parent_path is null then
		new_path := new_id::text::ltree;
		new_depth := 0;
	else
		new_path := (p_parent_path::text || '.' || new_id::text)::ltree;
    	new_depth := nlevel(p_parent_path);
	end if;

	insert into comments (id, user_id, post_id, path, depth, body, upvotes, downvotes)
	values (new_id, p_user_id, p_post_id, new_path, new_depth, p_body, p_upvotes, p_downvotes)
	returning * into result;

	return result;
end;

$$language plpgsql;