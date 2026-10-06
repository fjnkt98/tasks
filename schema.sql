CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE UNIQUE INDEX sessions_token_unique ON sessions (token);

CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT ('created'),
    created_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    updated_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) STRICT;

CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    digest TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
    updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;

CREATE UNIQUE INDEX users_name_unique ON users (name);
