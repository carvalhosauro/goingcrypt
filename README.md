# goingcrypt

A secure, self-hosted secret sharing platform built with Go. Share sensitive text snippets via encrypted, single-use links — with a full audit trail for administrators.

> Inspired by [foxcry.pt](https://foxcry.pt), extended with an admin panel, access logging, and a hybrid encryption model.

---

## How It Works

goingcrypt uses a **hybrid encryption model** designed so that even a compromised server cannot decrypt its own data:

1. The browser generates a random 256-bit key
2. The content is encrypted client-side using **AES-256-GCM** (Web Crypto API)
3. The server stores: `{ slug, SHA-256(key), ciphertext }` — never the key itself
4. The generated URL is split across two channels:
   - **Channel 1** (e.g. email): `https://goingcrypt.app/links/xK9mP2qR`
   - **Channel 2** (e.g. SMS): `#<decryption-key>`
5. On access, the server validates `SHA-256(key)`, logs the metadata, and invalidates the link
6. The browser decrypts the content locally — the key never reaches the server

```
Server knows:  slug + SHA-256(key) + ciphertext
Server never:  plaintext, raw key, or decrypted content
```

---

## Security Model

| Threat | Mitigated? | How |
|---|---|---|
| Compromised database | ✅ | Server never stores the decryption key |
| Malicious admin | ✅ | Ciphertext is useless without the key fragment |
| Link interception (in-transit) | ✅ | HTTPS + fragment never sent in HTTP requests |
| Link theft (before access) | ⚠️ | Mitigated by two-channel delivery |
| Link theft (after access) | ✅ | Links are single-use and immediately invalidated |
| Weak password storage | ✅ | argon2id with configurable memory cost |

---

## Features

- 🔒 Client-side AES-256-GCM encryption
- 🔗 Single-use, expiring links
- 📋 Admin panel with access audit logs (IP, User-Agent, timestamp)
- 👤 User authentication with MFA support
- 🚫 Link invalidation before access
- ⏱️ Lazy expiration + background janitor job
- 🧩 Hexagonal architecture — swap adapters without touching core logic

---

## Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Language | Go | Performance, concurrency, strong stdlib |
| Architecture | Hexagonal | Decoupled core, testable without infrastructure |
| Database | PostgreSQL | ACID transactions for atomic link invalidation |
| DB Access | sqlx | Balance between control and productivity |
| Client Crypto | Web Crypto API | Zero third-party dependencies in encryption layer |
| Password Hashing | argon2id | State-of-the-art KDF, memory-hard |
| Key Validation | SHA-256 | Sufficient for high-entropy random tokens |

---

## Architecture

```
/
├── internal/
│   ├── domain/          # Core entities and business rules
│   │   ├── link.go
│   │   └── user.go
│   ├── ports/           # Interfaces defined by the core
│   │   ├── repositories.go
│   │   └── services.go
│   └── core/            # Use cases
│       ├── link_service.go
│       └── auth_service.go
│
├── adapters/
│   ├── http/            # HTTP handlers
│   ├── postgres/        # Repository implementations
│   └── crypto/          # Crypto service implementation
│
└── cmd/
    └── api/
        └── main.go
```

---

## Database Schema

```sql
CREATE TYPE link_status AS ENUM ('WAITING', 'OPENED', 'EXPIRED');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password TEXT NOT NULL,          -- argon2id hash
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE  -- soft-delete
);

CREATE TABLE links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    slug VARCHAR(22) NOT NULL UNIQUE,    -- base62(uuid), public identifier
    hashed_key CHAR(64) NOT NULL UNIQUE, -- SHA-256 hex of decryption key
    ciphered_text TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    status link_status DEFAULT 'WAITING',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE link_access_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    link_id UUID UNIQUE NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    opened_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_links_status ON links(status);
CREATE INDEX idx_links_created_by ON links(created_by);
```

---

## API Endpoints

```
# Auth
POST   /api/v1/auth/signup
POST   /api/v1/auth/login
POST   /api/v1/auth/recovery
POST   /api/v1/auth/recovery/confirm

# Links (authenticated)
POST   /api/v1/links
GET    /api/v1/links/:slug
DELETE /api/v1/links/:slug

# Admin Panel (authenticated + admin role)
GET    /api/v1/admin/links
GET    /api/v1/admin/links/:slug
```

---

## Link Lifecycle

```
WAITING ──── accessed ──→ OPENED
   │
   ├── deleted by owner ──→ (hard delete)
   │
   └── expires_at passed ──→ EXPIRED (lazy + janitor cron)
```

---

## Getting Started

> 🚧 This project is under active development.

```bash
git clone https://github.com/you/goingcrypt
cd goingcrypt
cp .env.example .env
docker compose up -d
go run ./cmd/api
```

---

## Roadmap

- [x] Core encryption model
- [x] Database schema
- [x] API design
- [ ] Auth + MFA implementation
- [ ] Link creation and access flow
- [ ] Admin panel
- [ ] Background janitor job
- [ ] Optional: mandatory login for link access

---

## License

GNU General Public License version 2 (GPL-2.0)