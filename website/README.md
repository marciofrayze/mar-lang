# Belm Website (Elm + elm-ui)

This directory contains a standalone marketing/landing page for Belm.

## Run locally

```bash
cd website
elm make src/Main.elm --output=dist/app.js
python3 -m http.server 8080
```

Then open:

- `http://localhost:8080`

## Notes

- The page is intentionally separate from the compiler/runtime codebase.
- It is built with Elm + elm-ui and can be deployed as static files.
- Documentation links open an in-site Markdown viewer:
  - `docs/viewer.html?doc=getting-started`
  - `docs/viewer.html?doc=advanced`
- The markdown files in `website/docs/*.md` are the single source of truth.
