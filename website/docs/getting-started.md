# Belm Getting Started

This guide is intentionally short.

## 1. Create a `.belm` file

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

## 2. Compile

```bash
belm compile examples/todo.belm
```

## 3. Run

```bash
cd build/todo
./todo serve
```

Admin panel:

- `http://localhost:4100/_belm/admin`

## 4. Development mode

```bash
belm dev examples/todo.belm
```

## Next

- Read [advanced.md](./advanced.md)
- Open [../examples/todo.belm](../examples/todo.belm)
- Open [../examples/store.belm](../examples/store.belm)
