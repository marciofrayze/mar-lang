# Belm Language Support (VS Code)

This extension adds syntax highlighting, snippets/autocomplete, and basic language configuration for `.belm` files.

## Features

- Syntax highlighting for:
- Belm declarations (`app`, `entity`, `auth`, `rule`, `authorize`, `when`)
- Auth config keys (`user_entity`, `email_field`, etc.)
- Field modifiers (`primary`, `auto`, `optional`)
- Built-in types (`Int`, `String`, `Bool`, `Float`)
- Built-in functions (`contains`, `startsWith`, `endsWith`, `len`, `matches`, `isRole`)
- Comments (`--` and `#`)
- Strings, numbers, booleans, and operators
- Snippets/autocomplete (examples):
- `app`
- `entity`
- `field`
- `rule`
- `authorize`
- `auth`
- `authzcrud`

## Run Locally (Development Host)

1. Open this folder in VS Code:
   - `/Users/marcio/dev/github/belm/vscode-belm`
2. Press `F5` to start an Extension Development Host window.
3. Open any `.belm` file in the new window.

## Package for Installation

1. Install `vsce`:
   - `npm i -g @vscode/vsce`
2. From this folder, create a package:
   - `vsce package`
3. Install the generated `.vsix` in VS Code:
   - Command Palette -> `Extensions: Install from VSIX...`
