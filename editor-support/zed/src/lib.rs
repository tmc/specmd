use zed_extension_api::{self as zed, settings::LspSettings, LanguageServerId, Result};

struct SpecmdExtension;

impl zed::Extension for SpecmdExtension {
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
            .or_else(|| worktree.which("specmd-lsp"))
            .ok_or_else(|| "specmd-lsp not found; run go install ./cmd/specmd-lsp or set lsp.specmd-lsp.binary.path".to_string())?;
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

zed::register_extension!(SpecmdExtension);
