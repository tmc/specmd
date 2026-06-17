import * as vscode from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, Trace } from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration("specmd");
  const command = config.get<string>("lsp.path") || "specmd-lsp";
  const serverOptions: ServerOptions = { command, args: [] };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "markdown" }],
    synchronize: {},
  };

  client = new LanguageClient("specmd-lsp", "specmd LSP", serverOptions, clientOptions);
  const trace = config.get<string>("lsp.trace.server") || "off";
  if (trace === "messages") {
    client.setTrace(Trace.Messages);
  } else if (trace === "verbose") {
    client.setTrace(Trace.Verbose);
  }
  void client.start();
  context.subscriptions.push({ dispose: () => { void client?.stop(); } });

  context.subscriptions.push(vscode.commands.registerCommand("specmd.validateProject", async () => {
    await vscode.commands.executeCommand("workbench.action.problems.focus");
  }));
  context.subscriptions.push(vscode.commands.registerCommand("specmd.insertRequirement", insertRequirement));
  context.subscriptions.push(vscode.commands.registerCommand("specmd.openExtensionModel", openExtensionModel));
}

export function deactivate(): Thenable<void> | undefined {
  return client?.stop();
}

async function insertRequirement() {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "markdown") {
    return;
  }
  const snippet = new vscode.SnippetString("### Requirement: ${1:name}\n\nThe system SHALL ${2:behavior}.\n\n#### Scenario: ${3:name}\n\n- GIVEN ${4:context}\n- WHEN ${5:action}\n- THEN ${6:outcome}\n");
  await editor.insertSnippet(snippet, editor.selection.active);
}

async function openExtensionModel() {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) {
    return;
  }
  const picks = await vscode.workspace.findFiles("**/openspec/extensions/**/*.md", "**/node_modules/**", 50);
  if (picks.length === 0) {
    return;
  }
  const chosen = await vscode.window.showQuickPick(picks.map((uri) => ({
    label: vscode.workspace.asRelativePath(uri),
    uri,
  })));
  if (!chosen) {
    return;
  }
  await vscode.window.showTextDocument(chosen.uri);
}
