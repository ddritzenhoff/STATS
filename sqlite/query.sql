-- name: FindMemberByID :one
SELECT * FROM members
WHERE id = ? LIMIT 1;

-- name: FindMember :one
SELECT * FROM members
WHERE slack_uid = ? AND month_year = ? LIMIT 1;

-- name: CreateMember :one
INSERT INTO members (
    month_year,
    slack_uid
) VALUES (
    ?, ?
)
RETURNING *;

-- name: MostLikesReceived :one
SELECT m.*
FROM members m
WHERE month_year = ?
ORDER BY received_likes DESC
LIMIT 1;

-- name: MostDislikesReceived :one
SELECT m.*
FROM members m
WHERE month_year = ?
ORDER BY received_dislikes DESC
LIMIT 1;

-- name: UpdateMember :exec
UPDATE members
SET received_likes = ?,
received_dislikes = ?
WHERE id = ?;

-- name: DeleteMember :exec
DELETE FROM members
WHERE id = ?;
