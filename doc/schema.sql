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
