local telescope = require("telescope")
local builtin = require('telescope.builtin')
local actions = require('telescope.actions')

local fb_actions = telescope.extensions.file_browser.actions


require("telescope").setup {
    defaults = {
        mappings = {
            n = {
                ["q"] = actions.close,
            },
        },
        file_ignore_patterns = { "node%_modules/.*", ".git" },
    },

    extensions = {
        file_browser = {
            theme = "dropdown",
            grouped = true,
            hijack_netrw = true,
            mappings = {
                ["n"] = {
                    -- your custom normal mode mappings
                    ["N"] = fb_actions.create,
                    ["h"] = fb_actions.goto_parent_dir,
                    ["/"] = function()
                        vim.cmd("startinsert")
                    end,
                },
            },
        },
    },
}

telescope.load_extension("file_browser")

vim.keymap.set('n', '<C-p>', builtin.find_files, {})
vim.keymap.set('n', '<leader>ps', function()
    builtin.grep_string({ search = vim.fn.input("Grep > ") })
end)
vim.keymap.set('n', '<leader>vh', builtin.help_tags, {})
vim.keymap.set("n", "sf", function()
    telescope.extensions.file_browser.file_browser({
        path = "%:p:h",

        cwd = vim.fn.expand("%:p:h"),
        respect_gitignore = false,
        hidden = true,
        grouped = true,
        previewer = false,
        initial_mode = "normal",

        layout_config = { height = 40 },
    })
end)
