-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password,is_chirpy_red)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    FALSE
)
RETURNING id,created_at,updated_at,email,is_chirpy_red;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: GetAllChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetChirpsByAuthor :many
SELECT id, created_at, updated_at, body, user_id
FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetChirpByID :one
SELECT * FROM chirps WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1 AND user_id = $2;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password,is_chirpy_red
FROM users
WHERE email = $1;


-- name: StoreRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, created_at, updated_at, expires_at, revoked_at)
VALUES (
    $1, -- token
    $2, -- user_id
    NOW(), -- created_at
    NOW(), -- updated_at
    NOW() + INTERVAL '60 days', -- expires_at
    NULL -- revoked_at
)
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT user_id
FROM refresh_tokens
WHERE token = $1
AND expires_at > NOW()
AND revoked_at IS NULL;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE token = $1
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2,
    hashed_password = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, email;

-- name: UpgradeUserToChirpyRed :one
UPDATE users
SET is_chirpy_red = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, email, is_chirpy_red;
