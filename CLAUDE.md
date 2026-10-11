# CLAUDE.md

Context for AI assistants working in this repository. Read this before making changes.

## Project

University course project (đồ án): an e-commerce (book store) platform built as microservices behind an API gateway, with a recommendation service, deployed fully on Azure with monitoring. Grading favors a clean architecture, real cloud deployment, IaC, CI/CD and observability — keep those in mind when proposing solutions.

**Current status:** runs locally only (Docker Compose). `web`, `api-gateway` and `catalog-service` are implemented; `user-service` is a Nest.js scaffold; `order-service` is a Go scaffold (health check only); `recommender`, `infra/modules`, `infra/parameters` and `.github/workflows` are still empty (`.gitkeep`). Nothing is deployed to Azure yet. The owner writes the code incrementally — do NOT scaffold whole features unless explicitly asked; implement only the requested piece.

## Architecture decisions (agreed — don't change without asking)

| Concern | Decision | Why |
|---|---|---|
| Repo layout | Monorepo: one folder per service in `apps/`, plus `infra` | One repo for code, IaC and CI |
| Web | Next.js (UI); reads data from backend services through the API gateway | Frontend only talks to the gateway |
| API gateway | Go (`apps/api-gateway`), routes in `configs/routes.yaml`, port 8080 | Single public entry point for the backend |
| Catalog service | Go + Gin (`apps/catalog-service`), port 8083, migrations via `golang-migrate`, seed job `cmd/seed` | Books / subjects catalogue |
| User service | Nest.js 12 (`apps/user-service`), vitest, oxlint | Scaffolded, not implemented yet |
| Order service | Go + Gin (`apps/order-service`), port 8084, same layout as catalog-service, own DB `orders` | Orders and the server-side cart |
| Checkout & cart | Guest checkout allowed (`orders.user_id` nullable; guests look up orders by order code + email). Guest cart in localStorage; signed-in cart stored in order-service (`internal/cart`, tables `carts`/`cart_items`, no joins with `orders`), merged on login. Order creation takes `items[]` + an idempotency key and re-reads prices/stock from catalog-service. Payment gateway integration planned later | Cart can be split into its own service later by changing only the gateway upstream |
| Recommender | FastAPI (Python) separate service | ML ecosystem; not implemented yet |
| Functions | Azure Functions v4, Node.js/TypeScript, in `apps/functions` | Event-driven workloads; independently deployable |
| Database | SQL Server, **database per service** (e.g. `catalog`). On Azure: Azure SQL Database serverless, one logical server, one DB per service | Required: SQL Server on Azure; serverless is cheap |
| ORM (web) | Prisma 7 + `@prisma/adapter-mssql` | SQL Server support, migrations |
| DB access (recommender) | SQLAlchemy + pyodbc (ODBC Driver 18) | — |
| Cache | Redis (local: `redis:8-alpine`) | Not used by any service yet; Azure service not decided |
| Messaging | RabbitMQ (local only) | **Azure option not decided yet** — decide later (Azure has no managed RabbitMQ; candidates: Service Bus, or self-hosted container) |
| Hosting | Azure App Service Web Apps: `web` as code (Node.js runtime stack); backend services as containers | Required by the assignment: Web Apps |
| Registry | Azure Container Registry for all container images, pulled via managed identity | No admin credentials |
| Networking | `web` and `api-gateway` reachable from the internet; other services internal only | Only the gateway exposes the backend |
| Monitoring (local) | OpenTelemetry → Jaeger (traces), Prometheus + Grafana (metrics), Loki + Promtail (logs) — `infra/compose.yaml` | Local observability stack |
| Monitoring (Azure) | Application Insights (workspace-based) + Log Analytics + Azure Monitor metric alerts → email action group | Required by the course |
| IaC | Bicep, one module per concern in `infra/modules/` | Native Azure |
| CI/CD | GitHub Actions, Azure login via OIDC | No stored Azure secrets |
| Service-to-service | **gRPC** for calls that need an answer now; a message broker for everything that does not (broker choice still open, see Messaging). No HTTP between services. Contracts live in `proto/`; the generated Go code is committed in each service (`internal/pb`) | The gateway stays HTTP for browsers; gRPC stays inside the backend |
| Book reservation | Every book is a single copy (`AVAILABLE` / `RESERVED` / `SOLD`). Catalog serves `ReservationService` (`Reserve`, `Release`, `Sell`, `ReleaseExpired`) on gRPC port 9083 (`GRPC_PORT`). A hold records `reserved_by` (the order code) and `reserved_until`. Calls carry `x-internal-token` = `INTERNAL_TOKEN`; the gRPC server does not start without it | One UPDATE decides when two orders race for the same copy, all or nothing |
| Budget | Azure for Students credit ($100, 12 months; the subscription is disabled when it runs out). Prefer free tiers (e.g. the Azure SQL free offer, up to 10 free databases); say what a resource costs before adding a paid one | Cost decides infrastructure choices |

Service contracts: web calls the catalogue via `CATALOG_API_URL` (gateway, default `http://localhost:8080/api/catalog`); the gateway forwards to `CATALOG_SERVICE` (`http://catalog-service:8083`). Catalog service health: `GET /health`. The gateway forwards `/api/orders` (`auth_mode: optional`) and `/api/cart` (`auth_mode: required`) to `ORDER_SERVICE` (`http://order-service:8084`), both with `retry.attempts: 1` because gateway retries apply to every method. Order service health: `GET /health`. Planned: `GET {RECOMMENDER_URL}/api/v1/recommendations/{userId}?limit=N` → `[{ productId, score }]`.

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
apps/catalog-service/ Go: cmd/server, cmd/seed, internal/{book,grpcserver,pb,...}, migrations/, seed/books.jsonl.gz, Dockerfile, compose.yaml (own SQL Server)
apps/user-service/   Nest.js (scaffold)
apps/order-service/  Go: cmd/server, internal/{config,database,logger,router}, migrations/ (empty), Dockerfile, compose.yaml (own SQL Server)
apps/recommender/    FastAPI: app/{api,core,db,schemas,services,ml}, tests/, notebooks/ (empty)
apps/functions/      Azure Functions v4 (Node.js/TypeScript); function registrations under src/functions/
proto/               Protobuf contracts (catalog/v1/reservation.proto); Go code is generated into each service's internal/pb
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
- Order (DB → init → service): `docker compose up --build` in `apps/order-service`
- Gateway: `docker compose up --build` in `apps/api-gateway` (needs `configs/.env`)
- Whole stack: `docker compose up --build` at the repo root (includes infra, gateway, catalog, order)

Order service (run in `apps/order-service`): `go run ./cmd/server` (reads `configs/.env`), `go test ./...`

User service (run in `apps/user-service`): `npm run start:dev`, `npm test`, `npm run lint`

Recommender (run in `apps/recommender`, once implemented): `uvicorn app.main:app --reload`, `pytest`, `ruff check .`

Functions (run in `apps/functions`):
- `npm start` runs `clean` + `build` + `func start` (host on `http://localhost:7071`)
- `func new --template "<template>" --name <name>` adds a function; `func templates list` lists templates
- Core Tools: `npm i -g azure-functions-core-tools@4`

Protobuf / gRPC (run at the repo root; `protoc` and both Go plugins must be on PATH: `winget install Google.Protobuf`, then `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`):
- Generate: `protoc -I proto --go_out=apps/catalog-service/internal/pb --go_opt=paths=source_relative --go-grpc_out=apps/catalog-service/internal/pb --go-grpc_opt=paths=source_relative catalog/v1/reservation.proto`
- Re-run after every `.proto` change and commit the generated code (Docker builds do not run `protoc`).
- Try the catalog gRPC port: `grpcurl -plaintext -import-path proto -proto catalog/v1/reservation.proto -H "x-internal-token: <token>" -d '{}' localhost:9083 catalog.v1.ReservationService/ReleaseExpired`. Server reflection is off on purpose, so pass the `.proto`.

## Version gotchas — check docs, not memory

- **Next.js 16** has breaking changes. Read `apps/web/node_modules/next/dist/docs/` before writing Next.js code (see `apps/web/AGENTS.md`).
- **Prisma 7**: no `url` in the `datasource` block; the CLI URL lives in `prisma.config.ts`. `PrismaClient` requires a driver adapter: `new PrismaClient({ adapter: new PrismaMssql(process.env.DATABASE_URL) })`. Import the client from `@/generated/prisma/client`.
- Prisma dev tooling warns on Node 20; use Node 22.
- Azure SQL connection strings need `encrypt=true`; local SQL Server containers also need `trustServerCertificate=true`.
- The recommender Docker image must install `msodbcsql18` for pyodbc.
- **gRPC on Azure App Service** (Linux only): set HTTP version 2.0, HTTP 2.0 Proxy "gRPC only", and the app setting `HTTP20_ONLY_PORT` (a second port for HTTP/2). Calls must use TLS, client certificates are not supported (so no mTLS), and a gRPC call times out after 20 minutes idle. A custom container must still serve HTTP/1.1.
- **golang-migrate on SQL Server** sends each migration file as one batch, so a statement that names a column added earlier in the same file fails with "Invalid column name": wrap it in `EXEC(N'...')`. Avoid filtered indexes on tables edited from tools with `QUOTED_IDENTIFIER` off (sqlcmd without `-I`): every UPDATE on the table then fails with error 1934.

## Environment

- Never commit `.env` files, `node_modules`, or `*.log`. Only `.env.example` files are committed.

## Working rules

- Respond to the owner in Vietnamese; code, commits, file names and technical terms in English.
- Read the relevant files first; don't assume structure.
- Surgical changes only. Confirm before touching more than 3 files.
- Conventional commits (`feat/fix/chore/refactor/docs`). Propose branch name + commit message and wait for approval. Never push to `main`.
- The integration branch is `develop`: branch from `origin/develop` and open PRs into `develop`. `main` only holds the initial commit (switching to local `main` removes `apps/` from the working tree).
