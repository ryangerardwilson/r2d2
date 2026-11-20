## Context

The menu increments the cursor only while it is inside the screen list. The web screen adds the delta and then clamps. Both are the `j` and `k` path. Help returns before that path.

## Goals / Non-Goals

**Goals:**

- `j` and `k`, and the paired down and up keys, wrap the menu and every web list.
- An empty list does not move.
- Help still swallows `j` and `k`.

**Non-Goals:**

- Changing which screen Enter opens.
- A shared cursor package with c3po.
- The c3po screens and the nebula tunnel route list.

## Decisions

- Wrap with `(cursor + delta%n + n) % n` when the list has rows. Go's remainder keeps the sign of the dividend, so the added `n` keeps the index non-negative.
- `model.move` is the only web caller of a cursor step, and it is called from `j` and `k`. Wrapping there wraps every web list. `clamp` stays a clamp for the other assignments.
- The menu wraps in the `j` and `k` cases. Help still returns first.

## Risks / Trade-offs

- `k` on `web` now selects `slack`. The menu test that expected the cursor to stay on `web` has to expect `slack`.

## Migration Plan

- No stored data changes. An already-open menu keeps its current binary until it quits.
