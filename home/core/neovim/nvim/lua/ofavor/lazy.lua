-- [[ Install `lazy.nvim` plugin manager ]]
local lazypath = vim.fn.stdpath 'data' .. '/lazy/lazy.nvim'

if not (vim.uv or vim.loop).fs_stat(lazypath) then
  local lazyrepo = 'https://github.com/folke/lazy.nvim.git'
  local out = vim.fn.system { 'git', 'clone', '--filter=blob:none', '--branch=stable', lazyrepo, lazypath }
  if vim.v.shell_error ~= 0 then
    error('Error cloning lazy.nvim:\n' .. out)
  end
end ---@diagnostic disable-next-line: undefined-field

vim.opt.rtp:prepend(lazypath)

require('lazy').setup({
  'tpope/vim-sleuth', -- Detect tabstop and shiftwidth automatically

  require 'ofavor.plugins.gitsigns',
  require 'ofavor.plugins.whichkey',
  require 'ofavor.plugins.telescope',
  require 'ofavor.plugins.lazydev',
  require 'ofavor.plugins.lspconfig',
  require 'ofavor.plugins.conform',
  require 'ofavor.plugins.blinkcmp',
  require 'ofavor.plugins.theme',
  require 'ofavor.plugins.todo-comments',
  require 'ofavor.plugins.mini',
  require 'ofavor.plugins.treesitter',
  require 'ofavor.plugins.oil',
  require 'ofavor.plugins.neotree',
  require 'ofavor.plugins.ts-autotag',
  require 'ofavor.plugins.obsidian',
}, {
  ui = {
    -- If you are using a Nerd Font: set icons to an empty table which will use the
    -- default lazy.nvim defined Nerd Font icons, otherwise define a unicode icons table
    icons = vim.g.have_nerd_font and {} or {
      cmd = '⌘',
      config = '🛠',
      event = '📅',
      ft = '📂',
      init = '⚙',
      keys = '🗝',
      plugin = '🔌',
      runtime = '💻',
      require = '🌙',
      source = '📄',
      start = '🚀',
      task = '📌',
      lazy = '💤 ',
    },
  },
})
