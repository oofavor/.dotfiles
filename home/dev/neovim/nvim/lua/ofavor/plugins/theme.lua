-- tokyonight as the highlight engine, recolored with the system palette that
-- Nix writes to stdpath('data')/dusk-palette.lua. Falls back to stock colors.
local ok, p = pcall(dofile, vim.fn.stdpath 'data' .. '/dusk-palette.lua')

return {
  'folke/tokyonight.nvim',
  priority = 1000, -- Make sure to load this before all the other start plugins.
  config = function()
    ---@diagnostic disable-next-line: missing-fields
    require('tokyonight').setup {
      style = 'night',
      styles = {
        comments = { italic = false }, -- Disable italics in comments
        sidebars = 'dark',
        floats = 'dark',
      },
      on_colors = function(c)
        if not ok then
          return
        end
        c.bg = p.base00
        c.bg_dark = p.base01
        c.bg_dark1 = p.base01
        c.bg_float = p.base01
        c.bg_popup = p.base01
        c.bg_sidebar = p.base01
        c.bg_statusline = p.base01
        c.bg_highlight = p.base02
        c.bg_visual = p.base02
        c.bg_search = p.base03
        c.border = p.base02
        c.border_highlight = p.base0C

        c.fg = p.base05
        c.fg_dark = p.base04
        c.fg_float = p.base05
        c.fg_sidebar = p.base04
        c.fg_gutter = p.base03
        c.comment = p.base04
        c.dark3 = p.base03
        c.dark5 = p.base04
        c.terminal_black = p.base03

        c.blue = p.base0D
        c.blue0 = p.base02
        c.blue1 = p.base0C
        c.blue2 = p.base0C
        c.blue5 = p.base0D
        c.blue6 = p.base0C
        c.blue7 = p.base02
        c.cyan = p.base0C
        c.teal = p.base0C
        c.green = p.base0B
        c.green1 = p.base0C
        c.green2 = p.base0C
        c.magenta = p.base0E
        c.magenta2 = p.base08
        c.purple = p.base0E
        c.orange = p.base09
        c.yellow = p.base0A
        c.red = p.base08
        c.red1 = p.base08
      end,
      on_highlights = function(hl, c)
        if not ok then
          return
        end
        hl.CursorLineNr = { fg = p.base09, bold = true }
        hl.FloatBorder = { fg = p.base03, bg = c.bg_float }
        hl.WinSeparator = { fg = p.base02 }
        hl.TelescopeBorder = { fg = p.base03, bg = c.bg_float }
        hl.TelescopePromptBorder = { fg = p.base03, bg = c.bg_float }
        hl.TelescopePromptTitle = { fg = p.base00, bg = p.base09, bold = true }
        hl.TelescopePreviewTitle = { fg = p.base00, bg = p.base0C, bold = true }
      end,
    }

    vim.cmd.colorscheme 'tokyonight-night'
  end,
}
