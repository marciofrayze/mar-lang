# Belm

Belm is a declarative backend DSL inspired by [Elm](https://elm-lang.org) and [PocketBase](https://pocketbase.io), implemented in Go.

It is designed to be simple to read, version-control friendly, and fast to ship:

- declarative entities, rules, and authorization
- generated REST CRUD APIs
- built-in email-based login flow
- typed actions with atomic multi-entity writes
- embedded admin panel, monitoring, and SQLite backups
- embedded static frontend support

## Quick Start

### 1. Write a `.belm` app

```belm
app TodoApi
port 4100
database "todo.db"

entity Todo {
  id: Int primary auto
  title: String
  done: Bool

  rule "Title must have at least 3 chars" when len(title) >= 3
  authorize list when auth_authenticated
  authorize create when auth_authenticated
}
```

### 2. Compile

```bash
./belm compile examples/todo.belm
```

Output:

- executable: `build/todo/todo`
- generated clients:
  - `build/todo/clients/TodoApiClient.elm`
  - `build/todo/clients/TodoApiClient.ts`

### 3. Run

```bash
cd build/todo
./todo serve
```

Belm Admin is served from:

- `http://localhost:4000/_belm/admin`

### 4. Development mode (hot reload)

```bash
./belm dev examples/todo.belm
```

## Example: Typed Action

```belm
type alias PlaceOrderInput =
  { userId : Int
  , total : Float
  }

action placeOrder {
  input: PlaceOrderInput

  create Order {
    userId: input.userId
    total: input.total
    status: "created"
  }

  create AuditLog {
    userId: input.userId
    event: "order created"
  }
}
```

All `create` steps inside an action run in a single transaction.

## Commands

Belm CLI:

- `./belm compile <input.belm> [output-name]`
- `./belm dev <input.belm> [output-name]`
- `./belm format [--check] [--stdin] [files...]`
- `./belm lsp`
- `./belm version`

Generated app executable:

- `./<app> serve` (runs API + embedded admin)
- `./<app> backup` (creates SQLite backup)

## Main Concepts

- `entity`: schema + CRUD endpoint
- `rule`: validation logic
- `authorize`: per-operation authorization
- `auth`: email-code login configuration
- `action`: typed, transactional multi-entity write flow
- `system`: runtime/security/sqlite tuning
- `public`: static frontend embedding

## Examples

- [examples/todo.belm](/Users/marcio/dev/github/belm/examples/todo.belm)
- [examples/store.belm](/Users/marcio/dev/github/belm/examples/store.belm)

## Tooling

- Admin panel docs: [admin/README.md](/Users/marcio/dev/github/belm/admin/README.md)
- VS Code extension: [vscode-belm/README.md](/Users/marcio/dev/github/belm/vscode-belm/README.md)

## Full Documentation

For full reference and advanced topics, see:

- [docs/advanced.md](/Users/marcio/dev/github/belm/docs/advanced.md)

`docs/advanced.md` includes:

- full language syntax reference
- system/auth/public configuration details
- migration behavior
- monitoring, request logs, database tooling
- compiler architecture and diagrams
- generated clients details (Elm and TypeScript)
- LSP/formatter and extension notes
