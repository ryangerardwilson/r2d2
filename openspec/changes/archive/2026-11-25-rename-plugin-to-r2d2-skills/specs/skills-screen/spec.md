## MODIFIED Requirements

### Requirement: r2d2 skills keeps a clone of the skills repository

`r2d2 skills` SHALL keep a working copy of `https://github.com/ryangerardwilson/skills` at `~/.local/share/r2d2/skills`. The working copy SHALL NOT live under `~/.config/`. The skills SHALL be the directories under `plugins/r2d2-skills/skills/` that hold a `SKILL.md`. When the working copy is missing, or is an empty directory, r2d2 SHALL clone it. When it is that repository, r2d2 SHALL fetch `origin`. When the worktree is clean, HEAD is `main`, and `main` has no unpushed commits, r2d2 SHALL fast-forward `main` to `origin/main`. When the worktree is dirty, `main` has unpushed commits, or HEAD is not `main`, r2d2 SHALL NOT change the worktree, SHALL NOT reset, stash, or discard anything, and SHALL show that state. When the directory exists and is not that repository, r2d2 SHALL NOT delete or change it, SHALL show an error, and SHALL NOT open a writable list. Git SHALL run with prompts off.

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

`n` SHALL ask for a name. A name SHALL be lowercase letters, digits, and hyphens, SHALL NOT start or end with a hyphen, and SHALL be at most 64 characters. While that prompt is open, the screen SHALL show a steady block cursor in the cell immediately after the characters typed so far, and SHALL hide the cursor when the prompt closes. A bad name or a name that already exists SHALL be rejected with a message, SHALL NOT write a file, and SHALL leave the screen up. A good name SHALL write an empty `plugins/r2d2-skills/skills/<name>/SKILL.md` and SHALL open it in vim. When vim exits and the file is not empty, r2d2 SHALL commit `Add <name>` with the file exactly as vim left it. r2d2 SHALL NOT insert, require, or rewrite any header or frontmatter. When the file is empty, r2d2 SHALL remove the new directory and SHALL NOT commit.

Enter SHALL open the selected `SKILL.md` in vim. When the file changed, r2d2 SHALL commit `Update <name>` with the file exactly as vim left it. When it did not change, r2d2 SHALL NOT commit.

`d` SHALL ask for a confirmation. `y` SHALL remove the skill directory and commit `Remove <name>`. Any other key SHALL cancel and leave the list as it was.

`rn` SHALL ask for a new name of the selected skill. The field SHALL start as the current name, and a steady block cursor SHALL sit immediately after the characters typed so far. The name rules are the same as `n`. Esc SHALL cancel and SHALL NOT move the directory. The same name SHALL NOT commit. A good new name SHALL move the directory, SHALL keep the file bytes as they were, and SHALL commit `Rename <old> to <new>`.

`rr` SHALL fetch and fast-forward `main` when the worktree is clean, HEAD is `main`, and there are no unpushed commits, and SHALL otherwise say why it did not. `r` followed by any other key SHALL cancel.

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
- **THEN** `plugins/r2d2-skills/skills/deploy/SKILL.md` SHALL be exactly the text saved in vim
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

#### Scenario: A skill is renamed

- **WHEN** the operator presses `r` then `n` on `deploy`, changes the name to `ship`, and presses Enter
- **THEN** the directory SHALL be `ship`
- **AND** the file bytes SHALL be unchanged
- **AND** a commit `Rename deploy to ship` SHALL be pushed to `main`

#### Scenario: Rename cancel keeps the skill

- **WHEN** the operator presses `r`, then `n`, then Esc
- **THEN** the directory SHALL stay
- **AND** no commit SHALL be made

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



### Requirement: A push installs the skills for every host before the screen says pushed

After a push of `main` succeeds, r2d2 SHALL refresh the installed `r2d2-skills` plugin for Grok Build, Claude Code, and Codex, in that order. Add, update, rename, and remove SHALL all refresh the same way. For Grok Build, r2d2 SHALL run `grok plugin marketplace update skills` and then `grok plugin update r2d2-skills`. For Claude Code, r2d2 SHALL run `claude plugin marketplace update skills` and then `claude plugin update r2d2-skills@skills -y`. For Codex, r2d2 SHALL run `codex plugin marketplace upgrade skills` and then `codex plugin add r2d2-skills@skills --json`. These commands SHALL run with stdin closed, and their output SHALL NOT be painted over the screen.

A host's refresh SHALL succeed only when every directory where that host installed the plugin holds under `skills/` exactly the files of `plugins/r2d2-skills/skills/` at the commit just pushed, byte for byte, with no file missing and none extra. For Grok Build those directories SHALL be the `r2d2-skills` entry's `path` in `~/.grok/installed-plugins/registry.json` and, when it is local, its `source_path`. For Claude Code it SHALL be the `installPath` of `r2d2-skills@skills` in `plugins/installed_plugins.json` under the Claude Code config directory. For Codex it SHALL be the `installedPath` that `codex plugin add` printed. A command that exits 0 and leaves the old tree SHALL be a failure. A missing CLI, a plugin that is not installed, and a failed command SHALL each be a failure for that host and SHALL NOT be skipped.

The status SHALL say `pushed` only after all three hosts succeed, and SHALL then name the three hosts. When any host fails, the status SHALL be an error that names each failed host and its error, SHALL NOT say `pushed`, and SHALL NOT say the skill is installed. A failed refresh SHALL NOT roll back the commit or the push. r2d2 SHALL NOT edit a host's cache directory by hand, SHALL NOT change another plugin or marketplace, and SHALL NOT change host settings.

#### Scenario: A push installs for every host

- **GIVEN** the push of `Add deploy` succeeds
- **WHEN** Grok Build, Claude Code, and Codex each refresh and hold `deploy/SKILL.md` as it was pushed
- **THEN** the status SHALL start `pushed Add deploy`
- **AND** the status SHALL name Grok Build, Claude Code, and Codex
- **AND** while any refresh runs the status SHALL NOT say `pushed`

#### Scenario: A removed skill leaves every host

- **WHEN** the operator removes `deploy` and the push succeeds
- **THEN** each host's installed `skills/` SHALL NOT hold `deploy/SKILL.md`

#### Scenario: A failed refresh names the host and keeps the push

- **GIVEN** the push of `Add deploy` succeeds
- **WHEN** `claude plugin update r2d2-skills@skills -y` fails
- **THEN** the status SHALL be an error that names Claude Code and the command's error
- **AND** the status SHALL NOT say `pushed`
- **AND** `origin/main` and the local `main` SHALL keep `Add deploy`
- **AND** Grok Build and Codex SHALL still be refreshed

#### Scenario: A refresh that leaves the old files fails

- **GIVEN** the push of `Add deploy` succeeds
- **WHEN** a host's commands exit 0 and its installed `skills/` does not hold `deploy/SKILL.md`
- **THEN** the status SHALL name that host and `deploy/SKILL.md`
- **AND** the status SHALL NOT say `pushed`

#### Scenario: A missing host CLI is a failure

- **GIVEN** `codex` is not on PATH
- **WHEN** a push succeeds
- **THEN** the status SHALL name Codex and say `codex is not on PATH`
- **AND** the status SHALL NOT say `pushed`
