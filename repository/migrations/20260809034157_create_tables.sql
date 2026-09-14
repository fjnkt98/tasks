-- migrate:up
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  password TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;

CREATE TRIGGER trigger_users_updated_at AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = UNIXEPOCH() WHERE rowid == NEW.rowid;
END;

CREATE TABLE roles (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;

CREATE TABLE user_role_relations (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) STRICT, WITHOUT ROWID;

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (UNIXEPOCH()),
  updated_at INTEGER NOT NULL DEFAULT (UNIXEPOCH())
) STRICT;

CREATE TRIGGER trigger_tasks_updated_at AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET updated_at = UNIXEPOCH() WHERE rowid == NEW.rowid;
END;

-- migrate:down
DROP TRIGGER trigger_users_updated_at;
DROP TABLE users;
DROP TABLE roles;
DROP TABLE user_role_relations;
DROP TRIGGER trigger_tasks_updated_at;
DROP TABLE tasks;

