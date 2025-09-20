CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR UNIQUE NOT NULL,
  webauthn_user_handle BYTEA UNIQUE NOT NULL,
  profile_img_url VARCHAR,
  email VARCHAR UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Define enum for authenticator attachment
CREATE TYPE authenticator_attachment AS ENUM ('platform', 'cross-platform');

CREATE TABLE webauthn_credentials (
  id                       BYTEA PRIMARY KEY,  -- credentialId
  user_id                  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  public_key               BYTEA NOT NULL,     -- COSE key
  attestation_type         VARCHAR,            -- e.g. 'packed', 'fido-u2f', 'none'
  transports               JSONB NOT NULL,     -- array of supported transports
  user_present             BOOLEAN NOT NULL DEFAULT false,
  user_verified            BOOLEAN NOT NULL DEFAULT false,
  backup_eligible          BOOLEAN NOT NULL DEFAULT false,
  backup_state             BOOLEAN NOT NULL DEFAULT false,
  aaguid                   UUID NOT NULL,      -- 16-byte AAGUID stored as UUID
  sign_count               BIGINT NOT NULL DEFAULT 0, -- no uINT32 in PG, BIGINT is safe
  clone_warning            BOOLEAN NOT NULL DEFAULT false,
  authenticator_attachment authenticator_attachment NOT NULL,
  authenticator_data       BYTEA NOT NULL,
  public_key_algorithm     INTEGER NOT NULL,   -- COSE alg ID (-7, -257, etc.)
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at             TIMESTAMPTZ
);

CREATE TABLE sessions (
  id UUID PRIMARY KEY,
  user_id BIGINT NOT NULL,
  refresh_token VARCHAR NOT NULL,
  user_agent VARCHAR NOT NULL,
  client_ip VARCHAR NOT NULL,
  is_blocked BOOLEAN NOT NULL DEFAULT false,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title VARCHAR NOT NULL,
  topics JSONB,
  body JSONB NOT NULL,
  upvotes BIGINT NOT NULL DEFAULT 0,
  downvotes BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
  last_modified_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  post_id BIGINT NOT NULL,
  parent_id BIGINT,
  depth INT NOT NULL DEFAULT 0,
  upvotes BIGINT NOT NULL DEFAULT 0,
  downvotes BIGINT NOT NULL DEFAULT 0,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
  last_modified_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
  is_deleted bool NOT NULL DEFAULT false,
  deleted_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE post_votes (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  post_id BIGINT NOT NULL,
  vote SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
  last_modified_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

CREATE TABLE comment_votes (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  comment_id BIGINT NOT NULL,
  vote SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
  last_modified_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

COMMENT ON COLUMN post_votes.vote IS '1 for upvote, -1 for downvote';

COMMENT ON COLUMN comment_votes.vote IS '1 for upvote, -1 for downvote';

ALTER TABLE webauthn_credentials ADD FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE sessions ADD FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE posts ADD FOREIGN KEY (user_id) REFERENCES users (id);

ALTER TABLE comments ADD FOREIGN KEY (user_id) REFERENCES users (id);

ALTER TABLE comments ADD FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE;

ALTER TABLE comments ADD FOREIGN KEY (parent_id) REFERENCES comments (id) ON DELETE CASCADE;

ALTER TABLE post_votes ADD FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE comment_votes ADD FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE post_votes ADD FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE;

ALTER TABLE comment_votes ADD FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE;

-- added auto popularity column and its index to the comments
ALTER TABLE comments ADD COLUMN popularity BIGINT GENERATED ALWAYS AS (upvotes - downvotes) STORED;
