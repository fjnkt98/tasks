CREATE TABLE "schema_migrations" (version varchar(128) primary key);
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  password TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;
CREATE UNIQUE INDEX users_name_unique ON users (name);
CREATE TRIGGER trigger_users_updated_at AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = UNIXEPOCH() WHERE rowid == NEW.rowid;
END;
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT(UNIXEPOCH()),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;
CREATE UNIQUE INDEX sessions_token_index ON sessions (token);
CREATE TABLE statuses (
    name TEXT NOT NULL PRIMARY KEY
) STRICT, WITHOUT ROWID;
CREATE TABLE tasks (
  id INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  user_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  FOREIGN KEY (status) REFERENCES statuses(name) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;
CREATE TRIGGER trigger_tasks_updated_at AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET updated_at = UNIXEPOCH() WHERE rowid == NEW.rowid;
END;
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20260809033946'),
  ('20260809034157');
