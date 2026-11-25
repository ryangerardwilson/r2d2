# Rename the skills plugin to r2d2-skills

## Problem

Grok Build, Claude Code, and Codex list the shared skills as the plugin `skills`. That name does not say where the skills come from.

## Outcome

The marketplace stays `skills`. The one plugin is `r2d2-skills`, at `plugins/r2d2-skills/`. Skill files stay under that plugin's `skills/` directory. After a push, r2d2 refreshes `r2d2-skills` and leaves the marketplace name `skills`.

## Scope

- The skills repository manifests, plugin directory, and README.
- r2d2's clone path, refresh commands, help, and this spec.

## Non-goals

- Renaming the GitHub repository or the marketplace.
- Rewriting archived changes that described the old plugin name.
