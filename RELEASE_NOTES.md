# JTSO v1.1.0 — Release Notes

**Branch:** `optimize/code-improvements`  
**Base:** `main` (v1.0.19)  

---

## Highlights

This release introduces **on-demand telemetry collection**, **credential encryption at rest**, **Kafka output support**, **container monitoring**, and a comprehensive **settings management UI**, alongside significant code quality and security improvements.

---

## New Features

### On-Demand Telemetry Collection

A fully new subsystem for ad-hoc gNMI monitoring without modifying persistent profiles:

- Create, save, load, export, and manage on-demand monitoring profiles stored as JSON under `/var/ondemand/`.
- Per-field configuration: toggle monitoring, rate computation, float conversion, and tag inheritance.
- Start/stop on-demand collection at runtime — dynamically generates Telegraf configs, Grafana dashboards, and manages the `telegraf_ondemand` container lifecycle.
- Auto-generated Grafana dashboards with per-device filtering and dynamic variables via InfluxDB queries against an `ONDEMAND` measurement.
- Clear on-demand data by dropping the `ONDEMAND` InfluxDB measurement.
- New web portal page (`/ondemand.html`) and API endpoint (`POST /ondemandmgt`) with actions: `gnmionce`, `load`, `save`, `start`, `stop`, `clear`, `export`.

### gNMI Path Discovery

- New `GnmiOnDemand()` function subscribes to a gNMI path using SAMPLE mode and **auto-detects** fields (leaf nodes), tags/keys (XPath list keys), and aliases (common prefixes via a Trie-based algorithm).
- Returns structured `OnceReply` with discovered fields and aliases for use by the on-demand UI.
- Full XPath predicate-safe parsing with helper functions for complex path manipulation.

### Credential Encryption & Secret Management

- **AES-256-GCM encryption** for all stored NETCONF/gNMI passwords using the `APP_SECRET` environment variable.
- **Secret Manager** persists the secret to `/data/secret.txt` and supports seamless **key rotation**: detects a changed `APP_SECRET`, re-encrypts all credentials with the new key, and preserves the previous key during transition.
- **Auto-migration**: existing cleartext passwords are encrypted on first run (tracked via a new `passwordver` column).

### Kafka Output Support

- New `kafka_config` SQLite table storing: `enabled`, `brokers`, `topic`, `format`, `version`, `compression`, `messagesize`.
- Kafka configuration exposed in the Settings UI.
- `KafkaOutput` Telegraf template renders `[[outputs.kafka]]` stanzas with full configuration.
- When enabled, Kafka output is appended to both regular stack configs and on-demand configs.

### Telegraf Collector Tuning Parameters

- New `collector_parameters` SQLite table for `metric_batch_size`, `metric_buffer_limit`, `flush_interval`, `flush_jitter`.
- Runtime-configurable from the Settings page; propagated to all Telegraf instances via config file patching.
- Defaults: batch size 5000, buffer limit 100000, flush interval 5s, flush jitter 0s.

### Telemetry Interval Override System

- New `telegraf` SQLite table with a UNIQUE constraint on `(profile, path)` for per-profile, per-path interval overrides.
- CRUD operations via `POST /intervalmgmt` (actions: `getinterval`, `setinterval`, `reset`).
- Introspects profile definitions to list all gNMI subscription paths with default and configured intervals.
- Minimum interval enforced at 2 seconds.

### Container Monitoring & Logs

- `GetContainerStats()` collects CPU and memory usage for all running Docker containers using two-sample CPU delta calculation, stored in a thread-safe map.
- `GetContainerLogs()` retrieves the last 200 lines of stdout/stderr from a named container.
- New API endpoints: `GET /containerstats` and `GET /containerlogs?name=X`.
- Background ticker in `main.go` runs stats collection every minute.

### Stats & Settings Pages

- `GET /stats.html` — container statistics visualization page.
- `GET /settings.html` — expanded settings page with credentials, collector tuning, and Kafka configuration.
- `GET /pmanagement.html` — profile management page.

### InfluxDB Retention Policy Management

- New functions: `GetRetentionPolicyDuration()`, `AlterRetentionPolicyDuration()`, `RetentionDurationEqual()`.
- `DropMeasurement()` for clearing specific measurement data.
- Startup check compares current InfluxDB retention policy duration with the configured value and alters it if different.
- Default retention: `30d` (configurable from the admin table).

### SSE-Based gNMI Browser Streaming

- `GET /stream` — Server-Sent Events endpoint for real-time gNMI subscription data.
- `Streamer` object manages state, flush control, XPath deduplication counting, and stop signaling.
- Supports both legacy jstree (`TreeJs`) and new FancyTree (`FancytreeNode`) tree renderings.
- Client disconnect detection and cleanup.

---

## Improvements

### Package Restructuring

- `parser` package renamed to `gnmicollect` — consolidates all gNMI collection, tree building, and XPath parsing logic.
- New `security` package for encryption and secret management.
- New `ondemand` package for profile management and Grafana dashboard generation.

### Telegraf Config Optimization

- `OptimizeConf()` performs comprehensive multi-profile Telegraf configuration merging: deduplicates subscriptions, aliases, processors (clone, pivot, rename, enrichment, rate, converter, filtering, enum, regex, strings, monitoring), and outputs.
- gNMI subscription path optimization detects overlapping paths and keeps the shortest with the lowest interval.
- Unified `RenderConf()` generates complete Telegraf TOML from a `TelegrafConfig` struct.

### FancyTree Browser UI

- New `FancytreeNode` struct and `TraverseTreeFancytree()` / `PrintTreeFancytree()` for modern tree rendering.
- Configurable via `modules.portal.use_fancytree` and `modules.portal.hide_origin`.

### Expanded Device Family Support

- Now supports 13 device families: **mx, ptx, acx, ex, qfx, srx, crpd, cptx, vmx, vsrx, vjunos, vevo, ondemand** — each with independent Telegraf instances, debug mode, and path mappings.

### SQLite WAL Mode

- `PRAGMA journal_mode = WAL` enabled for better concurrent read/write performance.

### Graceful Shutdown

- Signal handling (`SIGINT`, `SIGTERM`, `SIGQUIT`) with context cancellation, ticker stops, DB close, and logger close.

---

## Security Enhancements

| Change | Detail |
|--------|--------|
| AES-256-GCM credential encryption | All stored NETCONF/gNMI passwords encrypted at rest using `APP_SECRET` env var |
| Secret rotation | Seamless key rotation with automatic re-encryption of all stored credentials |
| Path traversal prevention | On-demand file operations validate paths against the base directory |
| SQL injection prevention | `UpdateDebugMode()` validates instance names against an allowlist |
| Password migration | Auto-encrypts existing cleartext passwords on first run with encryption enabled |

---

## Code Quality & Optimization

| Category | Changes |
|----------|---------|
| Panic recovery | `HandlePanic` now uses `recover()` instead of `os.Exit(1)` |
| Context & shutdown | Propagated `context.Context` through long-running goroutines |
| Defer in loops | Extracted loop bodies with deferred closes into helper functions |
| Mutex patterns | `loadAllInternal()` extracted for safe nested locking; consistent `defer dbMu.Unlock()` |
| Template safety | Replaced runtime `template.Must` calls with init-time parsing |
| String building | Switched from repeated `+=` to `strings.Builder` in hot paths |
| Slice safety | Fixed potential slice header corruption with proper copies |
| Nil map guard | Added nil-map initialization checks before map writes |
| Docker client | Ensured `docker.Client.Close()` is called to prevent descriptor leaks |
| Error handling | Added checks for previously ignored errors across multiple packages |

---

## Database Schema Changes

**New tables:**

| Table | Purpose |
|-------|---------|
| `telegraf` | Per-profile, per-path telemetry interval overrides |
| `kafka_config` | Kafka output configuration |
| `collector_parameters` | Telegraf agent tuning parameters |

**Modified tables:**

| Table | New Columns |
|-------|-------------|
| `administration` | `ondemanddebug INTEGER`, `rpduration TEXT`, `ondemandconf TEXT` |
| `credentials` | `passwordver INTEGER` (0=cleartext, 1=encrypted) |

---

## New Web Portal Routes

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/stats.html` | Container statistics page |
| GET | `/settings.html` | Settings (credentials, tuning, Kafka) |
| GET | `/ondemand.html` | On-demand telemetry management |
| GET | `/pmanagement.html` | Profile management |
| GET | `/stream` | SSE endpoint for gNMI browser |
| GET | `/containerstats` | Container CPU/memory stats API |
| GET | `/containerlogs` | Container log retrieval API |
| POST | `/ondemandmgt` | On-demand profile CRUD + start/stop |
| POST | `/intervalmgmt` | Telemetry interval management |

---

## Configuration Changes

| Config Key | Type | Default | Purpose |
|------------|------|---------|---------|
| `modules.portal.use_fancytree` | bool | `true` | Use FancyTree UI for gNMI browser |
| `modules.portal.hide_origin` | bool | `true` | Strip origin prefix from gNMI paths |
| `modules.portal.browsertimeout` | int | `40` | gNMI browser subscription timeout (seconds) |
| `protocols.netconf.rpc_timeout` | int | `60` | NETCONF RPC timeout (seconds) |

---

## UI Changes

- **Added:** `ondemand.html`, `pmanagement.html`, `settings.html`, `stats.html`
- **Removed:** `cred.html`, `doc.html`
- **Added JS:** `ondemand.js`, `pmanage.js`, `settings.js`, `stats.js`, `browser2.js`
- **Removed JS:** `cred.js`, `doc.js`
- **CSS overhaul:** replaced `jtsostyle.css` with `jtsmain.css`; added Bootstrap Icons, jQuery UI, FancyTree stylesheets

---

## Breaking Changes

| Change | Impact |
|--------|--------|
| `APP_SECRET` env var required | Application requires `APP_SECRET` to be set for credential encryption |
| `parser` package renamed to `gnmicollect` | External references to the old package path will break |
| Credential storage format changed | Cleartext passwords are auto-migrated to encrypted format on first run; rollback without the secret key loses password access |
| Removed pages | `cred.html` and `doc.html` replaced by `settings.html` |

---

## File Summary

- **60 files changed** — 12,083 insertions, 2,374 deletions
- **5 new Go files** — `gnmicollect/parser.go`, `ondemand/ondemand.go`, `ondemand/grafana.go`, `security/crypto.go`, `security/secret_manager.go`
- **1 deleted Go file** — `parser/parser.go` (moved to `gnmicollect/`)
- **1 renamed Go file** — `parser/node.go` → `gnmicollect/node.go`
