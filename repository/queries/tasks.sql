-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = @id LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY id ASC LIMIT @limit OFFSET @offset;

-- name: ListTasksByStatus :many
SELECT * FROM tasks WHERE status = @status ORDER BY id ASC LIMIT @limit OFFSET @offset;

-- name: CreateTask :one
INSERT INTO tasks (title, description, status) VALUES (@title, @description, 'created') RETURNING *;

-- name: UpdateTask :exec
UPDATE tasks SET title = @title, description = @description, status = @status WHERE id = @id;

-- name: UpdateTaskStatus :exec
UPDATE tasks SET status = @status WHERE id = @id;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = @id;
