local M = {}

local function has_openspec_path(path)
  return path and path:find("/openspec/", 1, true) ~= nil
end

local function root_dir(bufnr)
  local name = vim.api.nvim_buf_get_name(bufnr)
  local root = vim.fs.root(bufnr, { "openspec", ".git" })
  if root and (has_openspec_path(name) or vim.uv.fs_stat(root .. "/openspec")) then
    return root
  end
  return nil
end

function M.setup(opts)
  opts = opts or {}
  local cmd = opts.cmd or { "openspec-lsp" }
  vim.api.nvim_create_autocmd("FileType", {
    pattern = "markdown",
    callback = function(args)
      local root = root_dir(args.buf)
      if not root then
        return
      end
      vim.lsp.start({
        name = "openspec-lsp",
        cmd = cmd,
        root_dir = root,
        capabilities = opts.capabilities,
        on_attach = opts.on_attach,
      }, { bufnr = args.buf })
    end,
  })
end

return M

