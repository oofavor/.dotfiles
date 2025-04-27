-- ONLY for NixOS due to its policy about binaries
-- LSP
vim.lsp.enable 'luals'
vim.lsp.config['luals'] = {
  cmd = { 'lua-language-server' },
  filetypes = { 'lua' },
  root_markers = { '.luarc.json', '.luarc.jsonc' },
}
