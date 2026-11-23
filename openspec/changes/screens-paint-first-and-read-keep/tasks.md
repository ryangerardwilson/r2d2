## 1. Screens

- [x] 1.1 web: live keys, one write queue, pending intents, site answer replaces the list, key read after the first paint
- [x] 1.2 looptab: background listings and session writer
- [x] 1.3 health: keep rows, retired rows replaced, keep vouches for idle objects

## 2. Verification

- [x] 2.1 `c3po ast exec -- openspec validate screens-paint-first-and-read-keep --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test ./...`
