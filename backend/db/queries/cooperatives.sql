-- name: GetCooperative :one
SELECT * FROM cooperatives
WHERE id = $1;
