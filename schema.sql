CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE UNIQUE INDEX sessions_token_unique ON sessions (token);
CREATE INDEX sessions_expires_at ON sessions (expires_at);

CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT ('created'),
    created_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    updated_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) STRICT;

CREATE INDEX tasks_user_id_status_id ON tasks (user_id, status, id DESC);

CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    digest TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
    updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;

CREATE UNIQUE INDEX users_name_unique ON users (name);

CREATE TABLE roles (
    name TEXT PRIMARY KEY
) STRICT, WITHOUT ROWID;

CREATE TABLE user_role_relations (
    user_id INTEGER NOT NULL,
    role TEXT NOT NULL,
    PRIMARY KEY (user_id, role),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (role) REFERENCES roles (name) ON DELETE CASCADE
) STRICT;
