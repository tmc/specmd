-- Minimal Neovim setup:
--
--   dofile("/path/to/openspec/editors/nvim/openspec-lsp.lua")
--
-- Or copy lua/openspec_lsp.lua into your runtimepath and call:
--
--   require("openspec_lsp").setup()

vim.opt.runtimepath:append(vim.fs.dirname(debug.getinfo(1, "S").source:sub(2)))
require("openspec_lsp").setup()
