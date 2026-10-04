-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = @id AND user_id = @user_id LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks WHERE user_id = @user_id ORDER BY status ASC, id DESC LIMIT @limit OFFSET @offset;

-- name: ListTasksByStatus :many
SELECT * FROM tasks WHERE user_id = @user_id AND status = @status ORDER BY status ASC, id DESC LIMIT @limit OFFSET @offset;

-- name: CreateTask :one
INSERT INTO tasks (title, status, user_id) VALUES (@title, 'created', @user_id) RETURNING *;

-- name: UpdateTask :one
UPDATE tasks SET title = @title, status = @status WHERE id = @id AND user_id = @user_id RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = @id AND user_id = @user_id;
