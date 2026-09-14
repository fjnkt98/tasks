-- name: GetUser :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY id ASC;

-- name: CreateUser :one
INSERT INTO users (name) VALUES (?) RETURNING id, name;

-- name: UpdateUser :exec
UPDATE users SET name = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = ? LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY id ASC;

-- name: CreateTask :one
INSERT INTO tasks (title, description, status) VALUES (?, ?, 'created') RETURNING *;

-- name: UpdateTask :exec
UPDATE tasks SET title = ?, description = ?, status = ? WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

