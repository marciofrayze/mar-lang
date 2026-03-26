# Mar Developer Tools

This extension adds syntax highlighting, snippets/autocomplete, and LSP features for `.mar` files.

Mar has a built-in `User` entity in every app, and entity operations are protected by default. The extension reflects that model in snippets and editor support.

## Features

- Syntax highlighting for Mar files
- Snippets and autocomplete for common Mar patterns
- Diagnostics, hover, go to definition, references, rename, symbols, and quick fixes
- Document formatting via `mar lsp`

## Install in VSCode

1. Open Extensions in VSCode.
2. Search for `Mar Developer Tools`.
3. Click `Install`.

If needed, set `mar.languageServer.path` in VSCode settings (examples: `mar`, `/abs/path/to/mar`).

## Format on Save

1. Open VS Code settings (`settings.json`) and configure:

```json
{
  "[mar]": {
    "editor.defaultFormatter": "mar-lang.mar-language-support",
    "editor.formatOnSave": true
  }
}
```

2. Save a `.mar` file to apply Mar formatting automatically.

## Notes

- Keep `mar` available in your `PATH` so the extension can start LSP and formatting.
