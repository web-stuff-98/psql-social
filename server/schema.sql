DROP DATABASE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(72) NOT NULL,
    /* "ADMIN" | "OWNER" | "USER" */
    role VARCHAR(5) NOT NULL,
    friends UUID[] DEFAULT '{}'::UUID[],
    blocked UUID[] DEFAULT '{}'::UUID[]
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(24) NOT NULL
);

CREATE TABLE room_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(24) NOT NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_channel_id UUID REFERENCES room_channels(id) ON DELETE CASCADE
);

CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE
);

CREATE TABLE comment_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE
);

CREATE TABLE post_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    room_channel_id UUID REFERENCES room_channels(id) ON DELETE CASCADE
);

CREATE TABLE profile_pictures (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  picture_data BYTEA NOT NULL
);

CREATE INDEX idx_username ON users (username);
CREATE INDEX idx_role ON users (role);
CREATE INDEX idx_friends ON users USING gin (friends);
CREATE INDEX idx_blocked ON users USING gin (blocked);
