.PHONY: build build-frontend build-backend dev dev-frontend dev-backend clean

build: build-frontend build-backend

build-frontend:
	cd frontend && npm run build

build-backend: build-frontend
	go build -ldflags "-s -w -X main.version=dev" -o bin/mybench .

dev:
	@echo "Run in separate terminals:"
	@echo "  make dev-backend"
	@echo "  make dev-frontend"

dev-backend:
	go run . _serve --no-browser

dev-frontend:
	cd frontend && npm run dev

clean:
	rm -rf bin/ frontend/dist/
