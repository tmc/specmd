use zed_extension_api::{self as zed, settings::LspSettings, LanguageServerId, Result};

struct OpenSpecExtension;

impl zed::Extension for OpenSpecExtension {
    fn new() -> Self {
        Self
    }

    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let binary = LspSettings::for_worktree(language_server_id.as_ref(), worktree)
            .ok()
            .and_then(|settings| settings.binary);
        let command = binary
            .as_ref()
            .and_then(|binary| binary.path.as_ref().map(|path| path.to_string()))
            .or_else(|| worktree.which("openspec-lsp"))
            .ok_or_else(|| "openspec-lsp not found; run go install ./cmd/openspec-lsp or set lsp.openspec-lsp.binary.path".to_string())?;
        let args = binary
            .and_then(|binary| binary.arguments)
            .unwrap_or_default();

        Ok(zed::Command {
            command,
            args,
            env: vec![],
        })
    }
}

zed::register_extension!(OpenSpecExtension);
