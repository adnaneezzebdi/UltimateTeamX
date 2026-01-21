# Identity Service – Database Schema

## Purpose

This database is dedicated exclusively to `identity`.
It is responsible only for identity and credentials.

It must NOT contain:
- business or gameplay logic
- roles or permissions
- gameplay flags
- refresh tokens
- authorization rules

If this schema grows beyond identity, the architecture is compromised.

---

## Isolation

- Dedicated database, not shared with any other service
- Used only by `identity`
- Other services must never connect to this database directly
- Communication with other services happens only via gRPC + JWT

This guarantees:
- better security for credentials
- clear separation of responsibilities
- independent migrations and backups

## Schema

`id`
  - UUID primary key
  - Not incremental
  - Prevents user enumeration attacks

`username`
  - Unique at database level
  - Required
  - Used only for identity, not for business logic

`email`
  - Unique at database level
  - Required
  - Used for login and recovery flows

`password_hash`
  - Secure hash using bcrypt or argon2
  - Never stored in plaintext
  - No custom crypto allowed

`created_at`
  - Timestamp of user creation
  - Default value: now()

## Architectural Decisions

These rules are non-negotiable:
- UUID as primary key
- UNIQUE constraints on username and email at DB level
- No extra fields beyond those defined
- No roles, permissions, flags, or tokens
- No runtime auto-migrations
- SQL migrations must be versioned

## Migrations

Database changes are managed through versioned SQL migrations.

Example:

`001_create_users.sql`

### Rules:
- Migrations are applied manually or via tooling
- No schema changes at application runtime
- Each migration must be atomic and ordered

### Security Rules
- Passwords are always hashed (bcrypt or argon2)
- Plaintext passwords are strictly forbidden
- No refresh tokens stored in DB
- No authorization data stored here
- Only identity data is allowed

### How to execute in terminal?
1. Create a file .env in service/identity/.env as .example.env
2. In terminal export the environmental variables: 
      `export $(cat service/identity/.env | xargs)`
3. Execute the migration:
      `PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f migrations/identity/001_create_users.sql`