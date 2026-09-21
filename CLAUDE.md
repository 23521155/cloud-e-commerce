# CLAUDE.md

Context for AI assistants working in this repository. Read this before making changes.

## Project

University course project (đồ án): an e-commerce platform with a recommendation service, deployed fully on Azure with monitoring. Grading favors a clean architecture, real cloud deployment, IaC, CI/CD and observability — keep those in mind when proposing solutions.

**Current status:** skeleton only. Empty folders contain `.gitkeep`. The owner writes the code incrementally — do NOT scaffold whole features unless explicitly asked; implement only the requested piece.

## Architecture decisions (agreed — don't change without asking)

| Concern | Decision | Why |
|---|---|---|
| Repo layout | Monorepo: `apps/web`, `apps/recommender`, `infra` | One repo for code, IaC and CI |
| Web | Next.js fullstack (UI + Route Handlers as backend) | No separate Node backend |
| Recommender | FastAPI (Python) separate service | ML ecosystem |
| Database | Azure SQL Database, serverless tier, shared by both services | Required: SQL Server on Azure; serverless is cheap |
| ORM (web) | Prisma 7 + `@prisma/adapter-mssql` | SQL Server support, migrations |
| DB access (recommender) | SQLAlchemy + pyodbc (ODBC Driver 18) | Reads same DB |
| Hosting | Azure App Service Web Apps: `web` deployed as code (Node.js runtime stack), `recommender` as a container | Required by the assignment: Web Apps |
| Registry | Azure Container Registry (recommender image only), pulled via managed identity | No admin credentials |
| Networking | `web` reachable from the internet; `recommender` internal only | Recommender is not public |
| Monitoring | Application Insights (workspace-based) + Log Analytics + Azure Monitor metric alerts → email action group | Required by the course |
| Telemetry SDKs | `@azure/monitor-opentelemetry` (web, via `src/instrumentation.ts`), `azure-monitor-opentelemetry` (recommender) | Official OpenTelemetry distros |
| IaC | Bicep, one module per concern in `infra/modules/` | Native Azure |
| CI/CD | GitHub Actions, Azure login via OIDC | No stored Azure secrets |

Service contract (planned): web calls `GET {RECOMMENDER_URL}/api/v1/recommendations/{userId}?limit=N` → `[{ productId, score }]`. Both services expose a health endpoint (`/api/health` for web, `/health` for recommender).

## Layout

```
apps/web/            Next.js 16 app (src/ dir, App Router, Tailwind 4, alias @/*)
  prisma/            schema.prisma (provider sqlserver); client generated to src/generated/prisma (gitignored)
  prisma.config.ts   Prisma CLI config, reads DATABASE_URL from .env
  src/app/(shop)     customer pages
  src/app/(admin)    admin pages
  src/app/api        Route Handlers
  src/lib            prisma client singleton, auth, recommender client
apps/recommender/    FastAPI: app/{api,core,db,schemas,services,ml}, tests/, notebooks/
infra/               Bicep: main.bicep, modules/, parameters/
.github/workflows/   CI (lint/test/build) and deploy pipelines
docs/                architecture notes and course report
scripts/             helper scripts
```

## Commands

Web (run in `apps/web`):
- `npm run dev` / `npm run build` / `npm run lint`
- `npm run db:migrate` (dev), `npm run db:deploy` (prod), `npm run db:studio`
- `postinstall` runs `prisma generate`

Recommender (run in `apps/recommender`, once implemented): `uvicorn app.main:app --reload`, `pytest`, `ruff check .`

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
