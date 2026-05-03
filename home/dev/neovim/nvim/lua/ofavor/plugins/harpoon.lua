return {
  'ThePrimeagen/harpoon',
  dependencies = { -- optional packages
    'nvim-lua/plenary.nvim',
  },
  config = function()
    local map = function(keys, func, desc, mode)
      mode = mode or 'n'
      vim.keymap.set(mode, keys, func, { desc = 'Harpoon: ' .. desc })
    end

    local terminal = require 'harpoon.term'
    local ui = require 'harpoon.ui'
    local navTo = function(num)
      return function()
        ui.nav_file(num)
      end
    end

    map('<leader>ja', require('harpoon.mark').add_file, 'Mark file')
    map('<leader>jf', ui.toggle_quick_menu, 'Toggle menu')
    map('<leader>jn', ui.nav_next, 'Next file')
    map('<leader>jp', ui.nav_prev, 'Prev file')
    map('<leader>jj', navTo(1), 'Go to 1')
    map('<leader>jk', navTo(2), 'Go to 2')
    map('<leader>jl', navTo(3), 'Go to 3')
    map('<leader>j;', navTo(4), 'Go to 4')
  end,
}
