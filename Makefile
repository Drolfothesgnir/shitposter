DB_URL=postgresql://root:secret@localhost:5432/shitposter?sslmode=disable

createdb:
	docker exec -it postgres17 createdb --username=root --owner=root shitposter

dropdb:
	docker exec -it postgres17 dropdb shitposter

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

dummy_comments:
	go clean -testcache
	go test -run TestCreateDummyComments ./...

db_schema:
	./generate_sql_schema.sh

.PHONY: new_migration db_schema migratedown migratedown1 migrateup migrateup1 sqlc test