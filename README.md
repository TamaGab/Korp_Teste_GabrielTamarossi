Containerized foundation for an Angular frontend and two independent Go microservices backed by PostgreSQL.

## Architecture

```text
Angular frontend
   |
   +----> Inventory Service ----> Inventory PostgreSQL
   |
   +----> Billing Service ------> Billing PostgreSQL
                |
                +----HTTP----> Inventory Service
```

Each microservice owns its database. The inventory service never accesses the billing database, and the billing service never accesses the inventory database. The billing service validates invoice Products through the inventory HTTP API using its Docker service name.

## Major directories

- `frontend/`: the Angular CLI-generated standalone application and its development Dockerfile.
- `backend/inventory-service/`: independent Go module for the inventory API and inventory database connection.
- `backend/billing-service/`: independent Go module for the billing API and billing database connection.
- `internal/database/`: PostgreSQL pool creation and automatic startup migrations in each service.
- `internal/health/`: the HTTP health handler in each service.
- `internal/product/`: Product and Available Stock behavior owned by the inventory service.
- `internal/invoice/`: Invoice creation behavior owned by the billing service.
- `cmd/api/`: each service's executable entry point, HTTP server, CORS configuration, and graceful shutdown.

## Requirements

- Docker
- Docker Compose
- Git

Go, Node.js, npm, Angular CLI, and PostgreSQL are not required on the host.

## Environment configuration

Compose includes safe development defaults so the stack starts immediately after cloning. To customize them, create a local environment file:

```bash
cp .env.example .env
```

The `.env` file is ignored by Git. Do not use development credentials in production.

## Run the application

Build and start all containers:

```bash
docker compose up --build
```

Stop and remove the application containers:

```bash
docker compose down
```

Stop the application and delete its persisted development database data and dependency caches:

```bash
docker compose down -v
```

Warning: `-v` permanently deletes the data in both development PostgreSQL volumes.

## URLs and ports

| Component            | URL or host port             |
| -------------------- | ---------------------------- |
| Frontend             | http://localhost:4200        |
| Inventory health     | http://localhost:8081/health |
| Billing health       | http://localhost:8082/health |
| Inventory PostgreSQL | localhost:5433               |
| Billing PostgreSQL   | localhost:5434               |

When the API and its database are available, both health endpoints return HTTP `200`:

```json
{
  "status": "ok",
  "service": "ok",
  "database": "ok"
}
```

When the API can respond but its database is unavailable, the endpoint returns HTTP `503`:

```json
{
  "status": "degraded",
  "service": "ok",
  "database": "unavailable"
}
```

## Useful Docker-only commands

Run Go tests:

```bash
docker compose exec inventory-service go test ./...
docker compose exec billing-service go test ./...
```

Update a service's Go module metadata after changing its dependencies:

```bash
docker compose exec inventory-service go mod tidy
docker compose exec billing-service go mod tidy
```

Run Angular commands inside the frontend container:

```bash
docker compose exec frontend npm test
docker compose exec frontend npm run build
```

Inspect service logs:

```bash
docker compose logs inventory-service billing-service
```

After editing Go source code, rebuild and recreate the affected service:

```bash
docker compose up --build --detach inventory-service
docker compose up --build --detach billing-service
```
