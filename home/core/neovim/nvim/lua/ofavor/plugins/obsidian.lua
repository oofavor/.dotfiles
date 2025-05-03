-- For obsidian.nvim
vim.opt.conceallevel = 1

vim.keymap.set('n', '<leader>ot', '<CMD>ObsidianTags<CR>', { desc = '[O]bsidian Tags' })
vim.keymap.set('n', '<leader>of', '<CMD>ObsidianTags<CR>', { desc = '[O]bsidian Tags' })

return {
  'epwalsh/obsidian.nvim',
  version = '*',
  lazy = true,
  ft = 'markdown',
  dependencies = { 'nvim-lua/plenary.nvim' },
  opts = {
    workspaces = {
      {
        name = 'Base',
        path = '~/obsidian/Base/',
      },
    },
    daily_notes = {
      folder = 'notes/dailies',
      default_tags = { 'daily-notes' },
    },
  },
}
