CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(72) NOT NULL,
    /* "ADMIN" | "USER" */
    role VARCHAR(5) NOT NULL,
    friends UUID [] DEFAULT '{}' :: UUID [],
    blocked UUID [] DEFAULT '{}' :: UUID []
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(24) NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE room_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(24) NOT NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(64) NOT NULL,
    content VARCHAR(5000) NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_channel_id UUID REFERENCES room_channels(id) ON DELETE CASCADE
);

CREATE TABLE room_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_channel_id UUID REFERENCES room_channels(id) ON DELETE CASCADE
);

CREATE TABLE direct_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content VARCHAR(200) NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE post_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES post_comments(id) ON DELETE CASCADE
);

CREATE TABLE direct_message_attachment_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bytes BYTEA NOT NULL,
    message_id UUID REFERENCES direct_messages(id) ON DELETE CASCADE,
    next_chunk UUID REFERENCES direct_message_attachment_chunks(id)
);

CREATE TABLE direct_message_attachment_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meta VARCHAR(128) NOT NULL,
    name VARCHAR(200) NOT NULL,
    size INT NOT NULL,
    first_chunk_id UUID REFERENCES direct_message_attachment_chunks(id),
    message_id UUID REFERENCES direct_messages(id) ON DELETE CASCADE
);

CREATE TABLE room_message_attachment_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bytes BYTEA NOT NULL,
    message_id UUID REFERENCES room_messages(id) ON DELETE CASCADE,
    next_chunk UUID REFERENCES room_message_attachment_chunks(id)
);

CREATE TABLE room_message_attachment_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meta VARCHAR(128) NOT NULL,
    name VARCHAR(200) NOT NULL,
    size INT NOT NULL,
    first_chunk_id UUID REFERENCES room_message_attachment_chunks(id),
    message_id UUID REFERENCES room_messages(id) ON DELETE CASCADE
);

CREATE TABLE post_comment_attachment_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bytes BYTEA NOT NULL,
    comment_id UUID REFERENCES post_comments(id) ON DELETE CASCADE,
    next_chunk UUID REFERENCES post_comment_attachment_chunks(id)
);

CREATE TABLE post_comment_attachment_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meta VARCHAR(128) NOT NULL,
    name VARCHAR(200) NOT NULL,
    size INT NOT NULL,
    first_chunk_id UUID REFERENCES post_comment_attachment_chunks(id),
    comment_id UUID REFERENCES post_comments(id) ON DELETE CASCADE
);

CREATE TABLE comment_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES post_comments(id) ON DELETE CASCADE
);

CREATE TABLE post_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE profile_pictures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    picture_data BYTEA NOT NULL
);

CREATE INDEX idx_username ON users (username);

CREATE INDEX idx_role ON users (role);

CREATE INDEX idx_friends ON users USING gin (friends);

CREATE INDEX idx_blocked ON users USING gin (blocked);