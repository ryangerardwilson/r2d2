## MODIFIED Requirements

### Requirement: The skills screen commits as the owner and pushes main only

Every commit SHALL have the author and committer Ryan Gerard Wilson <98247320+ryangerardwilson@users.noreply.github.com>. A commit SHALL hold only the paths of that skill. r2d2 SHALL push `main` to `origin/main` and SHALL NOT push another branch. r2d2 SHALL NOT force-push. When the push fails, the screen SHALL show the failure, the local commit SHALL stay, r2d2 SHALL NOT reset, stash, or discard anything, and r2d2 SHALL NOT refresh any host. When there is nothing to commit, r2d2 SHALL NOT push and SHALL NOT refresh any host.

#### Scenario: A refused push keeps the commit

- **GIVEN** the remote refuses the push
- **WHEN** the operator adds a skill
- **THEN** the screen SHALL say the push failed
- **AND** the local `main` SHALL keep the `Add <name>` commit
- **AND** no host refresh command SHALL run

#### Scenario: An empty commit refreshes nothing

- **GIVEN** a save leaves the skill's paths unchanged
- **WHEN** r2d2 commits
- **THEN** no push SHALL run
- **AND** no host refresh command SHALL run

## ADDED Requirements

### Requirement: A push installs the skills for every host before the screen says pushed

After a push of `main` succeeds, r2d2 SHALL refresh the installed `skills` plugin for Grok Build, Claude Code, and Codex, in that order. Add, update, rename, and remove SHALL all refresh the same way. For Grok Build, r2d2 SHALL run `grok plugin marketplace update skills` and then `grok plugin update skills`. For Claude Code, r2d2 SHALL run `claude plugin marketplace update skills` and then `claude plugin update skills@skills -y`. For Codex, r2d2 SHALL run `codex plugin marketplace upgrade skills` and then `codex plugin add skills@skills --json`. These commands SHALL run with stdin closed, and their output SHALL NOT be painted over the screen.

A host's refresh SHALL succeed only when every directory where that host installed the plugin holds under `skills/` exactly the files of `plugins/skills/skills/` at the commit just pushed, byte for byte, with no file missing and none extra. For Grok Build those directories SHALL be the `skills` entry's `path` in `~/.grok/installed-plugins/registry.json` and, when it is local, its `source_path`. For Claude Code it SHALL be the `installPath` of `skills@skills` in `plugins/installed_plugins.json` under the Claude Code config directory. For Codex it SHALL be the `installedPath` that `codex plugin add` printed. A command that exits 0 and leaves the old tree SHALL be a failure. A missing CLI, a plugin that is not installed, and a failed command SHALL each be a failure for that host and SHALL NOT be skipped.

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
- **WHEN** `claude plugin update skills@skills -y` fails
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
