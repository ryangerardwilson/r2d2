## 1. Screen

- [x] 1.1 List up to 20 recent recordings, newest first, under the six actions
- [x] 1.2 Enter on a recording plays it and does not send a camera command
- [x] 1.3 A missing player shows the error, and `q` leaves playback running

## 2. Check

- [x] 2.1 `openspec validate play-recent-recordings --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test ./internal/rec/`
