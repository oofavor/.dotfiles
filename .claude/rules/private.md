# Private Directory Access Restriction

Never read, list, search, or inspect `./private` or any files and subdirectories inside it.

- Do not use `Read`, `Grep`, `Glob`, or any other file tool on `./private` or paths under `./private/`.
- Do not use `Bash` commands (such as `ls`, `cat`, `head`, `tail`, `find`, `rg`, `grep`, `tree`, `python`, etc.) to view directory contents or read files inside `./private/`.
- Treat everything under `./private/` as strictly off-limits unless the user explicitly updates these rules.
