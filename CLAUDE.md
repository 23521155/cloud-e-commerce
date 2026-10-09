# CLAUDE.md

Context for AI assistants working in this repository. Read this before making changes.

## Project

University course project (đồ án): an e-commerce (book store) platform built as microservices behind an API gateway, with a recommendation service, deployed fully on Azure with monitoring. Grading favors a clean architecture, real cloud deployment, IaC, CI/CD and observability — keep those in mind when proposing solutions.

**Current status:** runs locally only (Docker Compose). `web`, `api-gateway` and `catalog-service` are implemented; `user-service` and `order-service` are Nest.js scaffolds; `recommender`, `infra/modules`, `infra/parameters` and `.github/workflows` are still empty (`.gitkeep`). Nothing is deployed to Azure yet. The owner writes the code incrementally — do NOT scaffold whole features unless explicitly asked; implement only the requested piece.

## Architecture decisions (agreed — don't change without asking)

| Concern | Decision | Why |
|---|---|---|
| Repo layout | Monorepo: one folder per service in `apps/`, plus `infra` | One repo for code, IaC and CI |
| Web | Next.js (UI); reads data from backend services through the API gateway | Frontend only talks to the gateway |
| API gateway | Go (`apps/api-gateway`), routes in `configs/routes.yaml`, port 8080 | Single public entry point for the backend |
| Catalog service | Go + Gin (`apps/catalog-service`), port 8083, migrations via `golang-migrate`, seed job `cmd/seed` | Books / subjects catalogue |
| User / Order services | Nest.js 12 (`apps/user-service`, `apps/order-service`), vitest, oxlint | Scaffolded, not implemented yet |
| Recommender | FastAPI (Python) separate service | ML ecosystem; not implemented yet |
| Functions | Azure Functions v4, Node.js/TypeScript, in `apps/functions` | Event-driven workloads; independently deployable |
| Database | SQL Server, **database per service** (e.g. `catalog`). On Azure: Azure SQL Database serverless, one logical server, one DB per service | Required: SQL Server on Azure; serverless is cheap |
| ORM (web) | Prisma 7 + `@prisma/adapter-mssql` | SQL Server support, migrations |
| DB access (recommender) | SQLAlchemy + pyodbc (ODBC Driver 18) | — |
| Cache | Redis (local: `redis:8-alpine`) | Azure service not decided yet |
| Messaging | RabbitMQ (local only) | **Azure option not decided yet** — decide later (Azure has no managed RabbitMQ; candidates: Service Bus, or self-hosted container) |
| Hosting | Azure App Service Web Apps: `web` as code (Node.js runtime stack); backend services as containers | Required by the assignment: Web Apps |
| Registry | Azure Container Registry for all container images, pulled via managed identity | No admin credentials |
| Networking | `web` and `api-gateway` reachable from the internet; other services internal only | Only the gateway exposes the backend |
| Monitoring (local) | OpenTelemetry → Jaeger (traces), Prometheus + Grafana (metrics), Loki + Promtail (logs) — `infra/compose.yaml` | Local observability stack |
| Monitoring (Azure) | Application Insights (workspace-based) + Log Analytics + Azure Monitor metric alerts → email action group | Required by the course |
| IaC | Bicep, one module per concern in `infra/modules/` | Native Azure |
| CI/CD | GitHub Actions, Azure login via OIDC | No stored Azure secrets |

Service contracts: web calls the catalogue via `CATALOG_API_URL` (gateway, default `http://localhost:8080/api/catalog`); the gateway forwards to `CATALOG_SERVICE` (`http://catalog-service:8083`). Catalog service health: `GET /health`. Planned: `GET {RECOMMENDER_URL}/api/v1/recommendations/{userId}?limit=N` → `[{ productId, score }]`.

## Layout

```
apps/web/            Next.js 16 app (src/ dir, App Router, Tailwind 4, alias @/*)
  prisma/            schema.prisma (provider sqlserver); client generated to src/generated/prisma (gitignored)
  prisma.config.ts   Prisma CLI config, reads DATABASE_URL from .env
  src/app/(shop)     customer pages
  src/app/(admin)    admin pages
  src/app/api        Route Handlers
  src/lib            prisma client singleton, auth, catalogue client (catalogue.ts)
apps/api-gateway/    Go: cmd/server, configs/{routes.yaml,.env}, deploy/{prometheus,loki}, Dockerfile, compose.yaml
apps/catalog-service/ Go: cmd/server, cmd/seed, migrations/, seed/books.jsonl.gz, Dockerfile, compose.yaml (own SQL Server)
apps/user-service/   Nest.js (scaffold)
apps/order-service/  Nest.js (scaffold)
apps/recommender/    FastAPI: app/{api,core,db,schemas,services,ml}, tests/, notebooks/ (empty)
apps/functions/      Azure Functions v4 (Node.js/TypeScript); function registrations under src/functions/
infra/               compose.yaml (shared local infra: redis, rabbitmq, jaeger, prometheus, grafana, loki, promtail)
  grafana/           Grafana provisioning + dashboards
  modules/, parameters/  Bicep for Azure (empty)
.github/workflows/   CI (lint/test/build) and deploy pipelines
docs/                architecture notes and course report
scripts/             helper scripts
```

## Commands

Web (run in `apps/web`):
- `npm run dev` / `npm run build` / `npm run lint`
- `npm run db:migrate` (dev), `npm run db:deploy` (prod), `npm run db:studio`
- `postinstall` runs `prisma generate`

Local stack (Docker Compose):
- Shared infra: `docker compose -f infra/compose.yaml up -d`
- Catalog (DB → init → migrate → seed → service): `docker compose up --build` in `apps/catalog-service`
- Gateway: `docker compose up --build` in `apps/api-gateway` (needs `configs/.env`)

User / Order services (run in `apps/<service>`): `npm run start:dev`, `npm test`, `npm run lint`

Recommender (run in `apps/recommender`, once implemented): `uvicorn app.main:app --reload`, `pytest`, `ruff check .`

Functions (run in `apps/functions`):
- `npm start` runs `clean` + `build` + `func start` (host on `http://localhost:7071`)
- `func new --template "<template>" --name <name>` adds a function; `func templates list` lists templates
- Core Tools: `npm i -g azure-functions-core-tools@4`

## Version gotchas — check docs, not memory

- **Next.js 16** has breaking changes. Read `apps/web/node_modules/next/dist/docs/` before writing Next.js code (see `apps/web/AGENTS.md`).
- **Prisma 7**: no `url` in the `datasource` block; the CLI URL lives in `prisma.config.ts`. `PrismaClient` requires a driver adapter: `new PrismaClient({ adapter: new PrismaMssql(process.env.DATABASE_URL) })`. Import the client from `@/generated/prisma/client`.
- Prisma dev tooling warns on Node 20; use Node 22.
- Azure SQL connection strings need `encrypt=true`; local SQL Server containers also need `trustServerCertificate=true`.
- The recommender Docker image must install `msodbcsql18` for pyodbc.

## Environment

- Never commit `.env` files, `node_modules`, or `*.log`. Only `.env.example` files are committed.

## Working rules

- Respond to the owner in Vietnamese; code, commits, file names and technical terms in English.
- Read the relevant files first; don't assume structure.
- Surgical changes only. Confirm before touching more than 3 files.
- Conventional commits (`feat/fix/chore/refactor/docs`). Propose branch name + commit message and wait for approval. Never push to `main`.
