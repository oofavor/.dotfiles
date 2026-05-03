-- For obsidian.nvim
vim.opt.conceallevel = 1

vim.keymap.set('n', '<leader>ot', '<CMD>ObsidianTags<CR>', { desc = '[O]bsidian Tags Search' })
vim.keymap.set('n', '<leader>od', '<CMD>ObsidianToday<CR>', { desc = '[O]bsidian Daily Note' })
vim.keymap.set('n', '<leader>of', '<CMD>ObsidianQuickSwitch<CR>', { desc = '[O]bsidian Notes Search' })
vim.keymap.set('n', '<leader>oi', '<CMD>ObsidianNew<CR>', { desc = '[O]bsidian Create Note' })
vim.keymap.set('n', '<leader>ob', '<CMD>ObsidianBacklinks<CR>', { desc = '[O]bsidian Backlinks' })

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
      folder = 'Daily',
      default_tags = { 'daily-notes' },
    },
    note_id_func = function(title)
      -- it might cause some unique primary key contrains but who cares
      return title
    end,
  },
}
