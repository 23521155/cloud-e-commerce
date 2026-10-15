# user-service

Accounts for the shop: registration, sign-in and sessions, the reader's profile, email confirmation, password reset, two-step sign-in (TOTP) and the staff role. Written in Go (Gin, sqlx), with its own SQL Server database `users`.

It is the *service đăng ký tài khoản người dùng* of `docs/course/rubric-project.csv` (4/Web). It talks to the purchase service (order-service) through an Azure Storage queue; see [Talking to order-service](#talking-to-order-service).

## Where it sits

```mermaid
flowchart LR
  browser([Browser]) --> web[web<br/>Next.js]
  web -->|/api/users/*| gw[api-gateway]
  gw -->|strips /api/users,<br/>verifies the JWT,<br/>sets X-User-*| us[user-service]
  us --> db[(SQL Server<br/>users)]
  us -->|outbox relay| q1[[email-outbox]]
  us -->|outbox relay| q2[[user-email-verified]]
  q1 --> fn[sendEmail<br/>Azure Function]
  fn --> mail[ACS Email / SMTP]
  q2 --> os[order-service]
```

- The gateway forwards `/api/users/<path>` to `user-service/<path>`. Example: `POST /api/users/auth/login` arrives here as `POST /auth/login`.
- The gateway also verifies the access token. On success it replaces any client-sent identity headers with `X-User-ID`, `X-User-Email` and `X-User-Roles`, taken from the token.
- user-service trusts `X-User-ID` and never parses a token itself. It must therefore only be reachable through the gateway: on Azure its ingress is internal.
- The web never sees tokens in page scripts. It keeps them in httpOnly cookies and calls the gateway from the server (`apps/web/src/lib/user-api.ts`, `session.ts`, `proxy.ts`).

## Run it locally

From the repository root:

```bash
docker compose up -d --build
```

This starts, among the rest:

| Container | What | Host port |
|---|---|---|
| `users-db` → `users-db-init` → `users-migrate` | SQL Server, the `users` database, migrations | `USERS_DB_PORT`, default 1436 |
| `user-service` | the API | 8085 |
| `azurite` | Azure Storage emulator (queues) | 10000–10002 |
| `functions` | Functions host running `sendEmail` | 7071 |
| `mailpit` | catches every email | UI at <http://localhost:8025>, SMTP 1025 |

Run the web app with `npm run dev` in `apps/web`, then open <http://localhost:3000/register>. The confirmation email shows up in Mailpit.

> On Windows, Hyper-V may reserve ports around 1436. If `users-db` cannot bind, set `USERS_DB_PORT` (for example `11436`) in the root `.env`.

Compose sets development defaults for every secret (`JWT_SECRET`, `MFA_ENCRYPTION_KEY`, the Azurite key). Do not reuse them anywhere else.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8085` | HTTP port |
| `DATABASE_URL` | — | `sqlserver://…?database=users&encrypt=true` (add `&trustServerCertificate=true` for a local container) |
| `JWT_SECRET` | — | Must equal the gateway's: it verifies the tokens signed here. The service does not start without it. |
| `JWT_ISSUER` | — | Must equal the gateway's |
| `JWT_ACCESS_TTL` | `15m` | Lifetime of an access token |
| `JWT_REFRESH_TTL` | `720h` | How long an unused device stays signed in |
| `WEB_URL` | `http://localhost:3000` | Base of the links in emails |
| `STORAGE_CONNECTION_STRING` | — | Storage account for the queues. Empty: messages wait in the outbox table. |
| `MFA_ENCRYPTION_KEY` | — | base64 of 32 random bytes (`openssl rand -base64 32`). Empty: two-step sign-in cannot be set up. Changing it makes existing TOTP secrets unreadable. |

`configs/.env.example` lists the same variables. For a run outside Docker, copy it to `configs/.env` (gitignored): it is read from the working directory when present.

## API

All bodies are JSON. Errors are `{"error": "<message>"}`. For a `400` caused by a bad field, the message names the field (for example `invalid input: password must be 8 to 128 characters`).

Endpoints marked 🔒 need a valid access token at the gateway; without one they answer `401`.

### Accounts and sessions

| Method and path | Body | Answers |
|---|---|---|
| `POST /auth/register` | `name`, `email`, `password` | `201` the account · `400` · `409` email taken |
| `POST /auth/login` | `email`, `password` | `200` tokens · `200 {mfaRequired, mfaToken, mfaExpiresAt}` when a code is due · `401` |
| `POST /auth/refresh` | `refreshToken` | `200` new tokens · `401` |
| `POST /auth/logout` | `refreshToken` | `204` always |
| 🔒 `GET /me` | — | `200` the account |
| 🔒 `PUT /me` | `name`, `phone`, `address`, `city`, `postalCode`, `country` | `200` the account · `400` |
| 🔒 `POST /me/password` | `current`, `password` | `200` tokens of a new session · `403` wrong current password · `400` |

**Tokens** (from login, refresh, password change and the MFA step):

```json
{
  "accessToken": "eyJ…",
  "tokenType": "Bearer",
  "expiresAt": "…",
  "refreshToken": "…",
  "refreshExpiresAt": "…",
  "user": { … }
}
```

**The account:**

```json
{
  "id": "0199…", "email": "…", "name": "…", "role": "customer",
  "emailVerifiedAt": null, "totpEnabledAt": null, "createdAt": "…",
  "phone": "", "address": "", "city": "", "postalCode": "", "country": ""
}
```

`id` is a UUIDv7 string. It is the JWT subject and `orders.user_id` in order-service. Empty profile strings mean "not given". The email cannot be changed: that would need the new address confirmed.

### Email links

| Method and path | Body | Answers |
|---|---|---|
| `POST /auth/verify-email` | `token` | `200 {email, emailVerifiedAt}` · `400` link used or expired |
| 🔒 `POST /me/verify-email` | — | `202` sent · `409` already confirmed · `429` one a minute |
| `POST /auth/forgot-password` | `email` | `202` always, account or not |
| `POST /auth/reset-password` | `token`, `password` | `204` · `400` link used or expired, or password refused |

### Two-step sign-in

| Method and path | Body | Answers |
|---|---|---|
| 🔒 `POST /me/mfa/totp/setup` | — | `200 {secret, otpauthUrl}` · `409` already on · `503` no key configured |
| 🔒 `POST /me/mfa/totp/confirm` | `code` | `200 {recoveryCodes}` (shown once) · `400` wrong code · `409` |
| 🔒 `POST /me/mfa/totp/disable` | `password`, `code` | `204` · `403` wrong password · `400` wrong code · `409` not on |
| `POST /auth/mfa/verify` | `mfaToken`, `code` | `200` tokens · `400` wrong code · `401` step expired or too many tries |

`code` is the 6 digits from the app, or a recovery code (`xxxx-xxxx-xxxx`, any case, dashes optional).

`GET /health` answers `200` when the database answers, `503` otherwise.

## How it works

### Passwords

- Hashed with argon2id: 19 MiB, 2 passes, 1 lane (the OWASP minimum). Stored in PHC format, so the parameters travel with each hash.
- Length is 8 to 128 characters.
- A wrong email and a wrong password get the same answer. Login also hashes when the email is unknown, so the response time does not reveal which accounts exist.

### Sessions

- **Access token:** JWT, HS256, signed with the gateway's secret. Claims are `sub`, `email`, `name`, `roles` and `amr`. `amr` is `["pwd"]`, or `["pwd","otp"]` after a second factor.
- **Refresh token:** 32 random bytes. Only its SHA-256 is stored, in `refresh_tokens`. Signing in starts a *family*: one row per refresh, one family per device. The family records the device name the gateway derives from the User-Agent, and its `amr`.
- **Rotation:** each refresh uses up the token and adds the next one to the family. A used token is accepted again for 30 seconds, because two tabs can refresh at once. After that, a used token means it was copied, and the whole family is revoked.
- **Sign-out** revokes the family. **Changing or resetting the password**, and **changing the role**, revoke every family of the account.
- Access tokens are not stored. One that was already issued stays valid until it expires (15 minutes by default), even after sign-out.

### Two-step sign-in (TOTP)

- RFC 6238: SHA-1, 6 digits, 30 seconds, as Google and Microsoft Authenticator read it from the QR code.
- **The secret:** setup creates it and stores it encrypted with AES-256-GCM (`MFA_ENCRYPTION_KEY`, package `internal/secretbox`). A first code turns two-step sign-in on and returns 10 recovery codes; only their hashes are kept.
- **Signing in:** with it on, the right password does not sign in. It opens a 5-minute *challenge* (`mfa_challenges`). `/auth/mfa/verify` finishes it with a code.
- **Codes:** the current 30-second step is accepted, plus one step either side. Each step is accepted once (`users.totp_last_step`). Five wrong codes end the challenge.

### Email, and the outbox

- **One transaction:** a change that must send something writes the message to the `outbox` table in the same transaction. Examples: a new account with its confirmation email, or a confirmed address with its event. Neither can happen without the other.
- **The relay** (`internal/outbox`) runs inside user-service. Every 2 seconds it claims up to 20 rows, puts each on its queue (base64 JSON, as Functions queue triggers expect), then deletes it.
  - A claim hides the row from other replicas for 30 seconds; a row that failed to send is claimed again after that.
  - Delivery is at least once, so every consumer must accept a message twice.
- **Writing the email:** `internal/email` writes each email (text and HTML).
- **Sending it:** the `sendEmail` Function (`apps/functions`) only delivers.
  - It is triggered by `email-outbox`.
  - On Azure it sends through Azure Communication Services; locally, over SMTP to Mailpit.
  - A send that fails is retried by the queue, and moves to `email-outbox-poison` after 5 tries.

| Link | Valid for | Notes |
|---|---|---|
| Confirm email (`/verify-email?token=…`) | 24 hours | At most one email a minute. The web page confirms only when the button is pressed, so a mail scanner that opens links cannot confirm. |
| Reset password (`/reset-password?token=…`) | 30 minutes | At most one a minute. Using one link uses up every reset link of the account. |

Only the SHA-256 of a link's token is stored (`email_tokens`).

### Talking to order-service

The shop allows guest checkout. When a reader confirms their address:

1. user-service writes a `user.email_verified` event to the outbox, in the same transaction as the confirmation. The relay puts it on the `user-email-verified` queue: `{"type":"user.email_verified","userId":"…","email":"…","verifiedAt":"…"}`.
2. order-service (`internal/accountevents`) reads the queue. It sets `user_id` on every guest order placed with that address (case ignored) and leaves orders that already belong to someone.

Handling the same message twice links nothing more. A message that keeps failing goes to `user-email-verified-poison`.

The address must be *confirmed* first, as WooCommerce also requires. Otherwise anyone could register someone else's email and see their orders.

### Staff

Staff are ordinary accounts with role `staff`. The shop office (`/admin` in the web) only lets in a session that has the staff role and passed the code step (`amr` contains `otp`). Give or take the role with `set-role`, which ships in the image and reads `DATABASE_URL`:

```bash
docker compose run --rm --no-deps --entrypoint ./set-role user-service -email someone@example.com -role staff
docker compose run --rm --no-deps --entrypoint ./set-role user-service -email someone@example.com -role customer
```

The account must exist (register it on the shop first). Its sessions end, so the next sign-in carries the new role.

## Database

Migrations live in `migrations/` and are applied by `migrate/migrate`: by compose locally, and by the deploy pipeline on Azure.

| Table | Holds |
|---|---|
| `users` | Account, argon2id hash, role, profile, `email_verified_at`, the sealed TOTP secret and its state |
| `refresh_tokens` | One row per refresh token (hash), its family, device, `amr`, used and revoked times |
| `email_tokens` | Confirmation and reset links (hash), with the address they were sent to |
| `outbox` | Messages waiting for their queue |
| `mfa_challenges` | The code step of a sign-in (hash), with its tries |
| `recovery_codes` | Recovery codes (hash), used once |

Expired tokens, challenges and used links are not cleaned up yet.

## Code layout

```
cmd/server        the API: wiring, the outbox relay, graceful shutdown
cmd/set-role      gives or takes the staff role
internal/user     handlers, service rules and SQL for accounts, sessions, links, TOTP
internal/token    signs access tokens (the gateway's claims, plus name and amr)
internal/outbox   outbox table, relay, queue sender, message shapes
internal/email    the emails' text and HTML
internal/secretbox AES-256-GCM for TOTP secrets
internal/config, database, logger, router   as in the other Go services
```

## Tests

```bash
go vet ./...
go test ./...

# Repository tests against a migrated database (the local container, for example):
DATABASE_URL='sqlserver://sa:Users_dev_123@localhost:1436?database=users&encrypt=true&trustServerCertificate=true' \
  go test -tags integration ./internal/user/
```

The unit tests replace the database, clock, token issuer and encryption with fakes. The integration tests create their own accounts and remove them afterwards.

## On Azure

- **user-service:** a Container App with internal ingress on port 8085 (`infra/modules/containerapps.bicep`), and the `users` database on the shared Azure SQL server.
- **sendEmail:** a Function App on the same Container Apps environment (`kind: functionapp`), scaled by Container Apps on the `email-outbox` queue.
- **Supporting resources:** the queues are in the storage account from `infra/modules/storage.bicep`, and email goes through Azure Communication Services (`infra/modules/communication.bicep`, Azure-managed domain).
- **Secrets:** `JWT_SECRET` and `MFA_ENCRYPTION_KEY` come from the repository's GitHub secrets.

The deploy pipeline (`.github/workflows/deploy.yml`) builds the images, runs the migrations and rolls the apps out on push to `develop`. Its smoke test calls `/api/users/health` through the gateway.
