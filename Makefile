.PHONY: dev dev-backend dev-frontend build build-backend build-frontend test clean

# Development: run frontend and backend concurrently
dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/sluff-server

dev-frontend:
	cd frontend && VITE_API_URL=http://localhost:8080 npm run dev

# Build
build: build-backend build-frontend

build-backend:
	cd backend && go build -o ../dist/sluff-server ./cmd/sluff-server

build-frontend:
	cd frontend && npm run build

# Test
test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test

# Clean
clean:
	rm -rf dist/
	rm -rf frontend/dist/
