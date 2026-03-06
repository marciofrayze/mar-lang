# Getting Started

## Install

1. **Download**: Get the latest binary from [github.com/marciofrayze/mar-lang/releases](https://github.com/marciofrayze/mar-lang/releases).
2. **Path**: Move `belm` to a directory in your `PATH` (`macOS/Linux`: `mv belm /usr/local/bin/belm && chmod +x /usr/local/bin/belm`; `Windows`: `setx PATH "%PATH%;C:\Tools\belm"`).
3. **Check**: `belm version`
4. **Code editor**: Belm currently supports only [VSCode](https://code.visualstudio.com/). Open Extensions (`Cmd+Shift+X` on macOS, `Ctrl+Shift+X` on Windows/Linux), search for `"Belm Language Support"`, and click Install. The extension requires `belm` on your `PATH` to start LSP and formatting.

## Quick Start

1. **Develop** with hot reload: `belm dev examples/store.belm`
2. **Compile** when ready to deploy: `belm compile examples/store.belm`
3. **Deploy**: `cd build/store && ./store serve`

## Create a `.belm` file

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

## Start development mode (hot reload)

```bash
belm dev examples/todo.belm
```

Belm rebuilds and restarts automatically whenever you save the file.

## Use the Admin UI while developing

Admin UI URL: `http://localhost:4100/_belm/admin`

1. Open **Authentication** and sign in.
2. Select an entity in the sidebar.
3. Use **New**, **Edit**, and **Delete** to test CRUD.
4. Use **Monitoring**, **Logs**, and **Database** for operational checks (admin role required).

## Build for deployment (final step)

```bash
belm compile examples/todo.belm
```

Output:

- executable: `build/todo/todo`
- generated clients:
  - `build/todo/clients/TodoApiClient.elm`
  - `build/todo/clients/TodoApiClient.ts`

## Deploy

Copy the generated binary to your server and start it:

```bash
./todo serve
```

Belm makes deployment straightforward: your entire backend ships as a single executable, including:

- API server
- authentication and authorization
- embedded static frontend assets (optional)
- embedded Admin UI
  - monitoring and performance dashboards
  - request logs dashboard
  - SQLite database backup tools

## Next

- Read the [Advanced Guide](./advanced.md)
- See [Examples](../examples/index.html)
