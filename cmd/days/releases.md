## Releases

### 1.1.0
Release Date: 2026-may-01
- Replaced panic-on-bad-input pattern in date helpers (`getDateInDays`, `getDaysSinceOrTo`, `getDaysBetween`) with error returns; main.go now exits with a one-line stderr message instead of a Go stack trace.
- Added local `die` helper in main.go.
