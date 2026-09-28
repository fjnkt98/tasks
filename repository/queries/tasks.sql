-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = @id LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY status ASC, id DESC LIMIT @limit OFFSET @offset;

-- name: ListTasksByStatus :many
SELECT * FROM tasks WHERE status = @status ORDER BY status ASC, id DESC LIMIT @limit OFFSET @offset;

-- name: CreateTask :one
INSERT INTO tasks (title, status) VALUES (@title, 'created') RETURNING *;

-- name: UpdateTask :one
UPDATE tasks SET title = @title, status = @status WHERE id = @id RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = @id;
