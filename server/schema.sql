DROP DATABASE;

DROP SCHEMA public CASCADE;

CREATE SCHEMA public;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(16) UNIQUE NOT NULL,
    password VARCHAR(72) NOT NULL,
    /* "ADMIN" | "USER" */
    role VARCHAR(5) NOT NULL,
    friends UUID [] DEFAULT '{}' :: UUID [],
    blocked UUID [] DEFAULT '{}' :: UUID []
);

CREATE TABLE friends (
    friender UUID REFERENCES users(id) ON DELETE CASCADE,
    friended UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (friender, friended)
);

CREATE TABLE friend_requests (
    friender UUID REFERENCES users(id) ON DELETE CASCADE,
    friended UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (friender, friended)
);

CREATE TABLE bios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content VARCHAR(300) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE blocks (
    blocked UUID REFERENCES users(id) ON DELETE CASCADE,
    blocker UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (blocked, blocker)
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(16) NOT NULL,
    private BOOLEAN NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE invitations (
    inviter UUID REFERENCES users(id) ON DELETE CASCADE,
    invited UUID REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (inviter, invited)
);

CREATE TABLE room_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(16) NOT NULL,
    main BOOLEAN NOT NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE TABLE room_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_channel_id UUID REFERENCES room_channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE direct_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content VARCHAR(200) NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bans (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, room_id)
);

CREATE TABLE members (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, room_id)
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

CREATE TABLE profile_pictures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    mime VARCHAR(10) NOT NULL,
    picture_data BYTEA NOT NULL
);

CREATE INDEX idx_username ON users (username);

CREATE INDEX idx_role ON users (role);

CREATE INDEX idx_friends ON users USING gin (friends);

CREATE INDEX idx_blocked ON users USING gin (blocked);