
## CREATE MIGRATION
migrate create -ext sql -dir database/migrations -seq create_verification_tokens_table

## RUN MIGRATION
migrate -database "postgres://postgres:postgres@localhost:5433/sayur_user_service?sslmode=disable" -path database/migrations up

## BERSIHKAN STATUS MIGRATION
migrate -path database/migrations -database "postgres://postgres:postgres@localhost:5433/sayur_user_service?sslmode=disable" force 1
