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
  const config = vscode.workspace.getConfiguration("belm");
  const configured = (config.get("languageServer.path") || "").trim();
  if (configured.length > 0) {
    return { command: configured, args: ["lsp"] };
  }

  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  if (workspaceFolder) {
    const belmFromWorkspace = path.join(workspaceFolder.uri.fsPath, "belm");
    if (fs.existsSync(belmFromWorkspace)) {
      return { command: belmFromWorkspace, args: ["lsp"] };
    }
    const belmFromParent = path.join(workspaceFolder.uri.fsPath, "..", "belm");
    if (fs.existsSync(belmFromParent)) {
      return { command: belmFromParent, args: ["lsp"] };
    }
  }

  return { command: "belm", args: ["lsp"] };
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
    documentSelector: [{ scheme: "file", language: "belm" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.belm"),
    },
  };

  client = new LanguageClient(
    "belmLanguageServer",
    "Belm Language Server",
    serverOptions,
    clientOptions
  );

  client.onDidChangeState((event) => {
    if (event.newState === State.Running) {
      reachedRunning = true;
    }
    if (event.newState === State.Stopped && !reachedRunning) {
      vscode.window.showErrorMessage(
        "Belm Language Server failed to start. Check belm.languageServer.path or ensure `belm` is available and supports `belm lsp`."
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
