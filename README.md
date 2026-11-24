# Cloud E-Commerce

An e-commerce platform built as Go microservices behind an API gateway, with a Next.js storefront and Azure Functions, deployed to Microsoft Azure.

## Layout

| Path | What it is |
|---|---|
| `apps/web` | Next.js storefront (Prisma, Azure SQL, Redis) |
| `apps/api-gateway` | Go API gateway, the single entry point for backend calls |
| `apps/catalog-service` | Go catalog service (HTTP + gRPC) |
| `apps/order-service` | Go order service |
| `apps/user-service` | Go user service |
| `apps/functions` | Azure Functions (TypeScript): emails, sell-photo ingestion |
| `proto/` | gRPC definitions |
| `infra/` | Bicep templates and local infrastructure (Compose) |
| `scripts/` | Load tests and test-data reset |
| `docs/` | Course brief and rubric |

## Run locally

Requires Docker, Node.js and Go.

```sh
cp apps/api-gateway/configs/.env.example apps/api-gateway/configs/.env
docker compose up -d --build
```

This starts the services, their SQL Server databases, Redis, RabbitMQ, Azurite and the observability stack.

| URL | Service |
|---|---|
| http://localhost:8080 | API gateway |
| http://localhost:8025 | Mailpit (local email) |
| http://localhost:3001 | Grafana |
| http://localhost:16686 | Jaeger |

Then run the web app, filling in `DATABASE_URL` in the copied `.env` before `npm run dev`:

```sh
cd apps/web
cp .env.example .env
npm install
npm run dev
```

## Deploy

GitHub Actions runs CI on every change (`.github/workflows/ci.yml`). A push to `main` deploys to Azure (`.github/workflows/deploy.yml`) using the Bicep templates in `infra/`.
