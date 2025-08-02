DB_URL=postgresql://root:secret@localhost:5432/shitposter?sslmode=disable

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

db_docs:
	dbdocs build doc/db.dbml

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

db_schema:
	echo "CREATE EXTENSION IF NOT EXISTS ltree;" > doc/schema.sql
	dbml2sql --postgres -o temp_schema.sql doc/db.dbml
	cat temp_schema.sql >> doc/schema.sql
	cat doc/indexes.sql >> doc/schema.sql
	cat doc/util.sql >> doc/schema.sql
	rm temp_schema.sql
	

.PHONY: new_migration db_docs db_schema migratedown migratedown1 migrateup migrateup1 sqlc test