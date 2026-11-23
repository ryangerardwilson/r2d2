## 1. Clone

- [x] 1.1 Clone a missing working copy, fetch and fast-forward a clean `main`, and leave a dirty clone, unpushed commits, or another branch as it is
- [x] 1.2 Refuse a directory that is not the skills repository without deleting it

## 2. Screen

- [x] 2.1 List each directory under `plugins/skills/skills/` that holds `SKILL.md`, with wrapping `j` and `k`, `g`, `G`, `?`, `q`, and `no skills`
- [x] 2.2 `n` checks the name, writes a minimal `SKILL.md`, opens vim, and commits `Add <name>`
- [x] 2.3 Enter opens vim and commits `Update <name>` when the file changed
- [x] 2.4 `d` confirms, removes the directory, and commits `Remove <name>`
- [x] 2.5 `r` fetches and fast-forwards, or says why it did not
- [x] 2.6 Push `main` without force, and keep a commit whose push failed

## 3. Menu and help

- [x] 3.1 The menu, `screen.Screens`, and `r2d2 help` list `skills`, `screen.Own` claims it, and `r2d2 skills help` explains it

## 4. Verification

- [x] 4.1 `openspec validate add-skills-screen --strict --no-interactive`
- [x] 4.2 `GOWORK=off go test ./...`
