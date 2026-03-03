
## CREATE MIGRATION
migrate create -ext sql -dir database/migrations -seq create_notifications_table

## RUN MIGRATION
migrate -database "postgres://postgres:postgres@localhost:5433/sayur_notification_service?sslmode=disable" -path database/migrations up

## BERSIHKAN STATUS MIGRATION
migrate -path database/migrations -database "postgres://postgres:postgres@localhost:5433/sayur_notification_service?sslmode=disable" force 1

## MUNDUR MIGRATION
migrate -path database/migrations -database "postgres://postgres:postgres@localhost:5433/sayur_notification_service?sslmode=disable" down 1