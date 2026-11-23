## Why

r2d2's own screens waited before they painted. `r2d2 web` dropped every key while a request ran, and a reorder needed three HTTPS round trips in a row before the new order appeared. `r2d2 looptab` wrote its session on every key and reloaded the job list on the key loop. `r2d2 health` listed two retired checks as `down` and could not see how far each application backup lagged; with nebula keep uploading only after a change, an idle database's object would also age into a false `down`.

## What Changes

- web: keys stay live while a request runs; writes go through one ordered queue; moves, deletes, and edits show at once as pending and are replaced by the site's answer, which the site now returns with the write; a failure reverts. The screen paints before it reads the API key.
- looptab: listings and the reload after Enter or `e` run in the background with a timeout; the session place is written in the background only when the job changes.
- health: reads `nebula keep status --json`; replaces `c3po-snapshot.timer` with `nebula-keep.timer` and `sqlx-backup.timer` with `nebula-sql-backup.timer`; drops `s3://sqlx-backups`; an object keep vouches for is not `down` for its age.

## Capabilities

### Modified Capabilities

- `r2d2`: the admin screen and the looptab screen.
- `health`: the checks.

## Impact

r2d2 still builds against its pinned c3po, nebula, and musica. The c3po screens (slack, gdrive, pass, o, cyclotree, deport) and nebula's tunnel screen reach r2d2 when those pins are bumped; `internal/health` imports `github.com/ryangerardwilson/nebula/s3`. The web changes need the site server that returns the list with each write; against the old server the screen reads the list with a GET as before.
