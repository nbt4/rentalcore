# TS Jobscanner Makefile

.PHONY: build run clean user-manager help

# Default target
all: build user-manager

# Build the main server
build:
	@echo "Building TS Jobscanner server..."
	go build -o server cmd/server/main.go

# Build the user management tool
user-manager:
	@echo "Building user manager..."
	go build -o user_manager user_manager.go

# Run the server
run: build
	@echo "Starting TS Jobscanner server..."
	./server

# Run in production mode
run-prod: build
	@echo "Starting TS Jobscanner server in production mode..."
	GIN_MODE=release ./server -config config.production.json

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f server user_manager

# Create first admin user
create-admin: user-manager
	@echo "Creating admin user..."
	./user_manager -username admin -email admin@tsunami-events.de -firstname Admin -lastname User -password "admin123"
	@echo "Default admin credentials: admin / admin123"
	@echo "Please change this password after first login!"

# List users
list-users: user-manager
	@echo "Listing all users..."
	./user_manager -list

# Test database connection
test-db: build
	@echo "Testing database connection..."
	timeout 5s ./server || echo "Database connection test completed"

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy

# Development setup
dev-setup: deps build user-manager
	@echo "Development setup complete!"
	@echo "Next steps:"
	@echo "1. Configure your database in config.json"
	@echo "2. Run 'make create-admin' to create your first user"
	@echo "3. Run 'make run' to start the server"

# Help
help:
	@echo "TS Jobscanner Build Commands:"
	@echo ""
	@echo "  build        - Build the main server binary"
	@echo "  user-manager - Build the user management tool"
	@echo "  run          - Build and run the server"
	@echo "  run-prod     - Run in production mode"
	@echo "  clean        - Remove build artifacts"
	@echo "  create-admin - Create an admin user interactively"
	@echo "  list-users   - List all users"
	@echo "  test-db      - Test database connection"
	@echo "  deps         - Install Go dependencies"
	@echo "  dev-setup    - Complete development setup"
	@echo "  help         - Show this help message"