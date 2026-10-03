-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at) VALUES (@user_id, @token, @expires_at) RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = @id LIMIT 1;

-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token = @token LIMIT 1;

-- name: UpdateSession :one
UPDATE sessions SET expires_at = @expires_at WHERE id = @id RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = @id;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions WHERE token = @token;
