DB_NAME = server/chat.db
SQL_INIT = server/database/init.sql

all: reset

reset:
	@echo "Deleting old database..."
	rm -f $(DB_NAME)
	@echo "Creating new database..."
	sqlite3 $(DB_NAME) < $(SQL_INIT)
	@echo "Database started successfully in the server folder!"