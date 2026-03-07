const fs = require("fs");
const path = require("path");
const vscode = require("vscode");
const {
  LanguageClient,
  State,
  TransportKind,
} = require("vscode-languageclient/node");

let client = null;
let reachedRunning = false;

function resolveServerOptions() {
  const config = vscode.workspace.getConfiguration("mar");
  const configured = (config.get("languageServer.path") || "").trim();
  if (configured.length > 0) {
    return { command: configured, args: ["lsp"] };
  }

  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  if (workspaceFolder) {
    const marFromWorkspace = path.join(workspaceFolder.uri.fsPath, "mar");
    if (fs.existsSync(marFromWorkspace)) {
      return { command: marFromWorkspace, args: ["lsp"] };
    }
    const marFromParent = path.join(workspaceFolder.uri.fsPath, "..", "mar");
    if (fs.existsSync(marFromParent)) {
      return { command: marFromParent, args: ["lsp"] };
    }
  }

  return { command: "mar", args: ["lsp"] };
}

function activate(context) {
  reachedRunning = false;
  const server = resolveServerOptions();

  const serverOptions = {
    command: server.command,
    args: server.args,
    transport: TransportKind.stdio,
  };

  const clientOptions = {
    documentSelector: [{ scheme: "file", language: "mar" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.mar"),
    },
  };

  client = new LanguageClient(
    "marLanguageServer",
    "Mar Language Server",
    serverOptions,
    clientOptions
  );

  client.onDidChangeState((event) => {
    if (event.newState === State.Running) {
      reachedRunning = true;
    }
    if (event.newState === State.Stopped && !reachedRunning) {
      vscode.window.showErrorMessage(
        "Mar Language Server failed to start. Check mar.languageServer.path or ensure `mar` is available and supports `mar lsp`."
      );
    }
  });

  context.subscriptions.push(client.start());
}

async function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

module.exports = {
  activate,
  deactivate,
};
