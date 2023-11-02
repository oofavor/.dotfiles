require("catppuccin").setup({
    flavour = "mocha",             -- latte, frappe, macchiato, mocha
    transparent_background = true, -- disables setting the background color.
    no_italic = true,              -- Force no italic
})

require("tokyonight").setup({
    style = "storm", -- The theme comes in three styles, `storm`, `moon`, a darker variant `night` and `day`
    transparent = true,
})

require("gruvbox").setup({
    transparent_mode = true,
})
-- setup must be called before loading
vim.cmd.colorscheme "catppuccin"
