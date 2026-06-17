-- Minimal Neovim setup:
--
--   dofile("/path/to/specmd/editor-support/nvim/specmd-lsp.lua")
--
-- Or copy lua/specmd_lsp.lua into your runtimepath and call:
--
--   require("specmd_lsp").setup()

vim.opt.runtimepath:append(vim.fs.dirname(debug.getinfo(1, "S").source:sub(2)))
require("specmd_lsp").setup()
