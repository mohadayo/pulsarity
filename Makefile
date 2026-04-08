.PHONY: up down build test test-python test-go test-ts lint lint-python lint-go lint-ts clean

# Docker Compose
up:
	docker compose up --build -d

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f

# Tests
test: test-python test-go test-ts

test-python:
	cd services/alert-manager && pip install -r requirements.txt -q && pytest -v

test-go:
	cd services/health-collector && go test -v ./...

test-ts:
	cd services/dashboard-api && npm install --silent && npm test

# Lint
lint: lint-python lint-go lint-ts

lint-python:
	cd services/alert-manager && pip install -r requirements.txt -q && flake8 --max-line-length=120 app.py test_app.py

lint-go:
	cd services/health-collector && go vet ./...

lint-ts:
	cd services/dashboard-api && npm install --silent && npx eslint 'src/**/*.ts'

# Cleanup
clean:
	docker compose down -v --rmi local
	rm -rf services/dashboard-api/node_modules services/dashboard-api/dist
