# Release Notes — v1.2.0

This release introduces a plugin framework with the new **JTT (Juniper Telemetry Test) plugin**, a **YANG Schema Explorer**, per-association **Kafka output control**, a refreshed UI, and a number of collection and stability fixes.

## Highlights

- **YANG Schema Explorer** — Download, browse, and inspect device YANG schemas from the web UI.
- **Per-association Kafka output** — Enable or disable Kafka output on a per-association basis instead of globally.
- **UI refresh** — New shared navigation bar, Inter web font, and consolidated styling.

## New Features

### YANG Schema Explorer

- New Schema page (`schema.html`) and supporting assets (`schema.js`, `netconf/schema.go`, `yangparser/yangparser.go`).
- Download YANG schemas per router over NETCONF with live progress streamed to the browser (Server-Sent Events), plus a **force re-download** option.
- Browse downloaded schema folders, list parsed schemas, and view individual schema content.
- New portal routes: `/schema.html`, `/downloadyang`, `/listschemas`, `/getschema`.

### Per-association Kafka output

- Kafka output can now be enabled/disabled per association rather than only globally.
- New `kafka` column on the `associations` table, with automatic migration for existing databases (`ALTER TABLE associations ADD COLUMN kafka ...`).
- Stack configuration now propagates the Kafka flag through collections, and Kafka output is only added when both the global Kafka config and the association flag are enabled.
- Collection logs now indicate whether Kafka output is enabled or disabled per collection.

## Improvements

- **On-demand subscriptions** now use `on_change` mode automatically when the interval is `0` (otherwise `sample`).
- **gNMI collection** respects client disconnects by checking a request context, avoiding wasted streaming after the client goes away, and handles the corner case where the base path is the leaf itself.
- Configurable **NETCONF RPC timeout** is now passed through to streaming and JTT job requests.
- **UI consolidation**: shared `navbar.html` template across all pages, new Inter web font (`myfonts.css`), and large `jtsmain.css` styling refresh.
- Default portal `browsertimeout` adjusted to `30` seconds.

## Configuration Changes

- `modules.portal.browsertimeout` default changed from `40` to `30`.

## Upgrade Notes

- The `associations` table is automatically migrated to add the new `kafka` column; existing associations default to Kafka output disabled (`no`). Review your associations after upgrade if you rely on Kafka output.
- No action is required for the JTT plugin unless you want to enable it — leave `plugins.jtt.url` empty to keep current behavior.

## Housekeeping

- Removed bundled Font Awesome font binaries and the unused `Navbar-Right-Links-icons.css` and `default.png` assets in favor of the new font/styling setup.
- Dependency updates in `go.mod` / `go.sum`.
- Version bumped to `1.2.0`.
