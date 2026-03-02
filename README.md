# Belm

Belm is an Elm-inspired language for backend development, implemented in Go, with a strong focus on readability, simplicity and maintainability.

## Goals

- Simple, declarative syntax (`entity`, `rule`, `authorize`, `auth`)
- Automatic REST CRUD
- SQLite as the database
- Email code login flow
- Rule-based authorization
- Safe automatic migrations

## Architecture (Go)

- [cmd/belmc/main.go](/Users/marcio/dev/github/belm/cmd/belmc/main.go): compiler/runtime CLI
- [internal/parser/parser.go](/Users/marcio/dev/github/belm/internal/parser/parser.go): `.belm` language parser
- [internal/expr/parser.go](/Users/marcio/dev/github/belm/internal/expr/parser.go): expression parser (`rule`/`authorize`)
- [internal/runtime](/Users/marcio/dev/github/belm/internal/runtime): HTTP server, auth/authz, and migrations
- [internal/sqlitecli/sqlitecli.go](/Users/marcio/dev/github/belm/internal/sqlitecli/sqlitecli.go): SQLite access via `sqlite3` binary (no external dependencies)

## Commands

Compile `.belm` into a JSON manifest:

```bash
go run ./cmd/belmc compile examples/store.belm build/store.manifest.json
```

When compiling, Belm also generates an Elm client in the same directory as the manifest.
Example output:

- `build/store.manifest.json`
- `build/StoreApiClient.elm`

Run directly from `.belm`:

```bash
go run ./cmd/belmc serve examples/store.belm
```

Run from a compiled manifest:

```bash
go run ./cmd/belmc serve-manifest build/store.manifest.json
```

## Auto-generated Elm Client

The generated module (`<AppName>Client.elm`) includes:

- `schema` (entity metadata)
- `rowDecoder`
- CRUD functions per entity:
- `list<Entity>`
- `get<Entity>`
- `create<Entity>`
- `update<Entity>`
- `delete<Entity>`
- auth endpoints, when auth is enabled:
- `requestCode`
- `login`
- `logout`
- `me`

Usage example in Elm:

```elm
import StoreApiClient as Api

type Msg
    = GotCustomers (Result Http.Error (List Api.Row))

load : Cmd Msg
load =
    Api.listCustomer
        { baseUrl = "http://localhost:4100", token = "" }
        GotCustomers
```

## Admin Panel

An Admin panel (built with Elm and elm-ui) is also provide:

- code: [admin/src/Main.elm](/Users/marcio/dev/github/belm/admin/src/Main.elm)
- docs: [admin/README.md](/Users/marcio/dev/github/belm/admin/README.md)

It uses `GET /_belm/schema` to discover entities and lets you list/create/update/delete records.

## VS Code Syntax Highlighting

A VS Code language extension for `.belm` files is available in:

- [vscode-belm](/Users/marcio/dev/github/belm/vscode-belm)

It provides syntax highlighting plus snippets/autocomplete for declarations, entities, auth/authorization blocks, rules, types, and common templates.

## Language Syntax

Minimal example:

```belm
app TodoApi
port 4000
database "./todo.db"

entity Todo {
  id: Int primary auto
  title: String
  done: Bool
  rule "Title must have at least 3 chars" when len(title) >= 3
}
```

### Statements

- `app <Name>`
- `port <number>`
- `database "<sqlite_path>"`
- `auth { ... }`
- `entity <Name> { ... }`

### Fields

`<fieldName>: <Type> [primary] [auto] [optional]`

Types:

- `Int`
- `String`
- `Bool`
- `Float`

Attributes:

- `primary`: primary key
- `auto`: auto-increment (usually with `Int primary`)
- `optional`: nullable field

If no primary key is provided, Belm automatically adds:

`id: Int primary auto`

## Business Rules (`rule`)

Inside `entity`:

```belm
rule "Customer must be 18 or older" when age >= 18
```

Operators:

- `and`, `or`, `not`
- `==`, `!=`, `>`, `>=`, `<`, `<=`
- `+`, `-`, `*`, `/`

Functions:

- `contains(text, part)`
- `startsWith(text, prefix)`
- `endsWith(text, suffix)`
- `len(value)`
- `matches(text, regex)`

Literals:

- `true`, `false`, `null`

If a rule fails, the API returns HTTP `422` with `error` and `details`.

## Authentication (`auth`)

Built-in email code login flow:

1. `POST /auth/request-code`
2. send the code by email
3. `POST /auth/login` (email + code) returns a bearer token
4. `POST /auth/logout` revokes the session

Configuration:

```belm
auth {
  user_entity Customer
  email_field email
  role_field role
  code_ttl_minutes 10
  session_ttl_hours 24
  email_transport console
  email_from "no-reply@store.local"
  email_subject "Your StoreApi login code"
  dev_expose_code true
}
```

`email_transport`:

- `console`: prints code in logs
- `sendmail`: uses local binary (`sendmail_path`)

## Authorization (`authorize`)

Per CRUD operation:

```belm
authorize list when isRole("admin")
authorize get when auth_authenticated and (id == auth_user_id or isRole("admin"))
authorize create when true
authorize update when auth_authenticated and (id == auth_user_id or isRole("admin"))
authorize delete when isRole("admin")
```

Context available in authorization expressions:

- `auth_authenticated`
- `auth_email`
- `auth_user_id`
- `auth_role`
- entity fields (`id`, `customerId`, etc.)

Extra function:

- `isRole("admin")`

## Generated Endpoints

For each entity `X`:

- `GET /xs`
- `GET /xs/:id`
- `POST /xs`
- `PUT /xs/:id`
- `PATCH /xs/:id`
- `DELETE /xs/:id`

Always:

- `GET /health`
- `GET /_belm/schema`

With auth enabled:

- `POST /auth/request-code`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`

## Migrations

Migrations run automatically on startup.

Automatic behavior:

- creates missing tables
- adds new optional columns
- creates/migrates internal auth tables
- records operations in `belm_schema_migrations`

Blocked (manual migration required):

- column type changes
- primary key changes
- nullability changes
- adding required fields to existing tables
- adding primary/auto columns to existing tables

When blocked, the server fails at startup with a clear error message.

## Full Example

Use [examples/store.belm](/Users/marcio/dev/github/belm/examples/store.belm), which already includes:

- business rules (`age >= 18`, email validation, etc.)
- email code auth
- role/ownership authorization
- entities: `Customer`, `Product`, `Order`, `OrderItem`
