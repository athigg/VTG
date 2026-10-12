# VTG Store

A server-rendered vintage marketplace built with Go, SQLite, GORM, and `html/template`.

## Requirements

- Go 1.27 or later
- No C compiler is required; SQLite uses a pure-Go driver.

## Run locally

```powershell
$env:APP_ENV = "development"
go run ./cmd/server
```

Open <http://localhost:8080>. On first run, the app creates `data/vtg-store.db`, applies the schema, and seeds sample listings and demo accounts. Uploaded JPEG/PNG product photos are stored under `static/uploads/` (maximum 5 MB).

The app reads configuration from environment variables. `.env.example` lists supported values; it is a reference file and is not automatically loaded.

## Local demo accounts

These fixed credentials exist only when `APP_ENV=development` (the default for local runs):

| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin@vtg.local` | `VintageAdmin123!` |
| Shopper | `demo@vtg.local` | `VintageUser123!` |

The sign-in page offers fill-in links for these accounts only in development. Passwords are stored as bcrypt hashes. Seed accounts are created only if the email does not already exist; the seeder never overwrites an existing account's password or role. To override the demo admin account on its first seed, set `DEMO_ADMIN_EMAIL` and `DEMO_ADMIN_PASSWORD` before starting the app.

## Production safety

Set `APP_ENV=production`. Demo users, demo products, and demo-admin prefill are disabled. Set independent `SESSION_KEY` and `CSRF_KEY` values (each at least 32 characters) using your platform's cryptographic random generator. Never use the documented local demo passwords or development fallback keys in production. `Secure` cookies and HSTS are enabled in production; terminate HTTPS at the app or a trusted TLS reverse proxy.

Admin roles can be assigned from the dashboard. For production, provision the first administrator through a controlled database operation or one-time secure administrative process; do not enable demo seeding.

## Features

- Registration, login/logout, signed cookie sessions, bcrypt password hashes
- CSRF protection and security response headers
- Searchable/category-filtered listings; owner-scoped create, edit, and delete
- JPEG/PNG upload validation, generated filenames, and upload size limits
- Profile editing and admin controls for account role/status and listings
- SQLite schema migration and idempotent local sample data

Run tests with `go test ./...`.