-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    token,
    user_id,
    expires_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, NOW(), NOW()
);

-- name: GetUserFromRefreshToken :one
SELECT
  token,
  user_id,
  expires_at,
  revoked_at
FROM
  refresh_tokens
WHERE
  token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = $1, updated_at = $2
WHERE token = $3;

-- name: UpdateUser :one
UPDATE users
SET email = $2,
    hashed_password = $3,
    updated_at =  NOW()
WHERE id = $1
RETURNING *;

-- name: UpgradeUserToChirpyRed :exec
UPDATE users SET is_chirpy_red = TRUE, updated_at = NOW() WHERE id = $1;
