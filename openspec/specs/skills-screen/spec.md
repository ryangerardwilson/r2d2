# skills-screen Specification

## Purpose
r2d2 skills edits the shared skills for Claude Code, Grok Build, and Codex in a clone of ryangerardwilson/skills and pushes each change to main, which every host installs as the one skills plugin.
## Requirements
### Requirement: r2d2 skills keeps a clone of the skills repository

`r2d2 skills` SHALL keep a working copy of `https://github.com/ryangerardwilson/skills` at `~/.local/share/r2d2/skills`. The working copy SHALL NOT live under `~/.config/`. The skills SHALL be the directories under `plugins/skills/skills/` that hold a `SKILL.md`. When the working copy is missing, or is an empty directory, r2d2 SHALL clone it. When it is that repository, r2d2 SHALL fetch `origin`. When the worktree is clean, HEAD is `main`, and `main` has no unpushed commits, r2d2 SHALL fast-forward `main` to `origin/main`. When the worktree is dirty, `main` has unpushed commits, or HEAD is not `main`, r2d2 SHALL NOT change the worktree, SHALL NOT reset, stash, or discard anything, and SHALL show that state. When the directory exists and is not that repository, r2d2 SHALL NOT delete or change it, SHALL show an error, and SHALL NOT open a writable list. Git SHALL run with prompts off.

#### Scenario: A missing clone is cloned

- **GIVEN** `~/.local/share/r2d2/skills` does not exist
- **WHEN** the operator opens `r2d2 skills`
- **THEN** r2d2 SHALL clone the repository there
- **AND** the screen SHALL list its skills

#### Scenario: A dirty worktree is left in place

- **GIVEN** the working copy has an uncommitted change
- **WHEN** the screen opens or the operator presses `r`
- **THEN** the change SHALL still be there
- **AND** `main` SHALL NOT move
- **AND** the screen SHALL say the worktree is dirty

#### Scenario: Another directory is refused

- **GIVEN** `~/.local/share/r2d2/skills` holds files that are not the skills repository
- **WHEN** the operator opens `r2d2 skills`
- **THEN** those files SHALL stay
- **AND** the screen SHALL show an error
- **AND** `n`, Enter, and `d` SHALL NOT write

### Requirement: The skills screen lists, creates, edits, and deletes skills

On a terminal, `r2d2 skills` SHALL list the skills by name. `j` and `k` SHALL move and SHALL wrap. `g` and `G` SHALL jump to the first and last skill. An empty list SHALL leave the cursor in place and SHALL say `no skills`. The screen SHALL NOT write shortcut text until `?`, and a second `?` SHALL hide it. `q` SHALL quit. The screen SHALL paint only in black, white, and gray.

`n` SHALL ask for a name. A name SHALL be lowercase letters, digits, and hyphens, SHALL NOT start or end with a hyphen, and SHALL be at most 64 characters. While that prompt is open, the screen SHALL show a steady block cursor in the cell immediately after the characters typed so far, and SHALL hide the cursor when the prompt closes. A bad name or a name that already exists SHALL be rejected with a message, SHALL NOT write a file, and SHALL leave the screen up. A good name SHALL write an empty `plugins/skills/skills/<name>/SKILL.md` and SHALL open it in vim. When vim exits and the file is not empty, r2d2 SHALL commit `Add <name>` with the file exactly as vim left it. r2d2 SHALL NOT insert, require, or rewrite any header or frontmatter. When the file is empty, r2d2 SHALL remove the new directory and SHALL NOT commit.

Enter SHALL open the selected `SKILL.md` in vim. When the file changed, r2d2 SHALL commit `Update <name>` with the file exactly as vim left it. When it did not change, r2d2 SHALL NOT commit.

`d` SHALL ask for a confirmation. `y` SHALL remove the skill directory and commit `Remove <name>`. Any other key SHALL cancel and leave the list as it was.

`r` SHALL fetch and fast-forward `main` when the worktree is clean, HEAD is `main`, and there are no unpushed commits, and SHALL otherwise say why it did not.

A missing vim SHALL say `vim is not on PATH`, SHALL NOT write a file, and SHALL leave the screen up. Writes SHALL be off when HEAD is not `main`.

#### Scenario: The name prompt uses a block cursor

- **WHEN** the operator presses `n`
- **THEN** a steady block cursor SHALL sit immediately after `new skill: `
- **WHEN** the operator types `ab`
- **THEN** the block cursor SHALL sit immediately after `ab`
- **WHEN** the operator presses Esc
- **THEN** the cursor SHALL be hidden

#### Scenario: A new skill is added

- **WHEN** the operator presses `n`, types `deploy`, presses Enter, and saves the file in vim
- **THEN** `plugins/skills/skills/deploy/SKILL.md` SHALL be exactly the text saved in vim
- **AND** a commit `Add deploy` SHALL be pushed to `main`
- **AND** the list SHALL show `deploy`

#### Scenario: A bad or taken name is rejected

- **WHEN** the operator types `Deploy`, `-deploy`, or the name of a skill that exists
- **THEN** the screen SHALL say why
- **AND** no file SHALL be written
- **AND** the screen SHALL stay up

#### Scenario: An edit commits only a change

- **WHEN** the operator presses Enter on `deploy` and quits vim without a change
- **THEN** no commit SHALL be made
- **WHEN** the operator presses Enter again and saves a change
- **THEN** a commit `Update deploy` SHALL be pushed to `main`

#### Scenario: A skill is removed

- **WHEN** the operator presses `d` on `deploy` and then `y`
- **THEN** the directory SHALL be gone
- **AND** a commit `Remove deploy` SHALL be pushed to `main`

#### Scenario: Cancel keeps the skill

- **WHEN** the operator presses `d` and then `n`
- **THEN** the skill SHALL stay
- **AND** no commit SHALL be made

#### Scenario: The empty list

- **GIVEN** no skill exists
- **WHEN** the screen opens
- **THEN** the screen SHALL say `no skills`
- **AND** `j` and `k` SHALL leave the cursor in place

### Requirement: The skills screen commits as the owner and pushes main only

Every commit SHALL have the author and committer Ryan Gerard Wilson <98247320+ryangerardwilson@users.noreply.github.com>. A commit SHALL hold only the paths of that skill. r2d2 SHALL push `main` to `origin/main` and SHALL NOT push another branch. r2d2 SHALL NOT force-push. When the push fails, the screen SHALL show the failure, the local commit SHALL stay, and r2d2 SHALL NOT reset, stash, or discard anything.

#### Scenario: A refused push keeps the commit

- **GIVEN** the remote refuses the push
- **WHEN** the operator adds a skill
- **THEN** the screen SHALL say the push failed
- **AND** the local `main` SHALL keep the `Add <name>` commit

### Requirement: r2d2 skills explains itself off a terminal

`r2d2 skills help` SHALL print the skills help and SHALL exit 0. When stdin or stdout is not a terminal, `r2d2 skills` SHALL print the same help and SHALL exit 0 without cloning. `--resume <id>` SHALL be accepted. Other arguments SHALL exit 2.

#### Scenario: A pipe prints the help

- **WHEN** `r2d2 skills` runs with stdout on a pipe
- **THEN** the text SHALL name `r2d2 skills` and `~/.local/share/r2d2/skills`
- **AND** the command SHALL exit 0

