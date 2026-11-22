-- Solve LeetCode problems in Neovim: `leet` from the shell (alias for `nvim leetcode.nvim`)
-- or `:Leet` inside a session. The first launch asks for the Cookie request header
-- copied from a logged-in leetcode.com tab (DevTools > Network).
local leet_arg = 'leetcode.nvim'

return {
  'kawre/leetcode.nvim',
  -- Load at startup only for `nvim leetcode.nvim`, otherwise wait for :Leet.
  lazy = vim.fn.argv(0, -1) ~= leet_arg,
  cmd = 'Leet',
  dependencies = {
    'nvim-telescope/telescope.nvim',
    'nvim-lua/plenary.nvim',
    'MunifTanjim/nui.nvim',
  },
  opts = {
    arg = leet_arg,
    lang = 'cpp', -- switch per question with :Leet lang
    picker = { provider = 'telescope' },
    -- Lets :Leet start with other buffers open (it takes its own tab; leave with :Leet exit).
    plugins = { non_standalone = true },
    hooks = {
      question_enter = {
        function(q)
          local map = function(keys, cmd, desc)
            vim.keymap.set('n', keys, '<CMD>Leet ' .. cmd .. '<CR>', { buffer = q.bufnr, desc = 'Leetcode: ' .. desc })
          end
          map('<leader>lr', 'run', '[R]un Testcases')
          map('<leader>ls', 'submit', '[S]ubmit')
          map('<leader>ld', 'desc', 'Toggle [D]escription')
          map('<leader>lc', 'console', '[C]onsole')
          map('<leader>ll', 'lang', 'Change [L]anguage')
          map('<leader>lt', 'tabs', 'Question [T]abs')
          map('<leader>lh', 'hints', '[H]ints')
          map('<leader>li', 'info', '[I]nfo')
          map('<leader>lo', 'open', '[O]pen in Browser')
          map('<leader>lm', 'menu', '[M]enu')
        end,
      },
    },
  },
}
