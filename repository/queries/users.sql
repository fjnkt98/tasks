-- name: CreateUser :exec
INSERT INTO users (name, password) VALUES (@name, @password);

-- name: GetUser :one
SELECT * FROM users WHERE id = @id LIMIT 1;

-- name: GetUserByName :one
SELECT * FROM users WHERE name = @name LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE users SET password = @password WHERE id = @id;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id;
