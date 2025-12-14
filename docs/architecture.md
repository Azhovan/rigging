# Architecture Overview

This document outlines Rigging's internal structure to help contributors understand how configuration flows from sources into strongly typed structs.

## High-Level Components

- **Loader**: Orchestrates the full lifecycle of loading configuration, including source ordering, merging, strict key checking, binding to structs, tag-based validation, custom validators, provenance storage, and optional watch support for reloads. Later sources override earlier ones, and strict mode rejects unknown keys before binding occurs.【F:loader.go†L12-L106】【F:loader.go†L145-L186】
- **Sources**: Implement the `Source` interface to provide normalized key/value pairs and an optional watch stream. `SourceWithKeys` can additionally return original keys to enrich provenance for env/file inputs.【F:types.go†L9-L28】
- **Binding**: Converts normalized map data into struct fields while tracking per-field provenance. Binding errors are combined with validation results to produce a single `ValidationError`.【F:loader.go†L70-L112】【F:provenance.go†L1-L32】
- **Validation**: Runs built-in tag validation (e.g., required, default, min/max, oneof) followed by user-supplied validators that implement the `Validator` interface. All validation feedback is aggregated so callers receive every field issue at once.【F:loader.go†L104-L140】【F:types.go†L39-L55】
- **Watch Loop**: The `Watch` API performs an initial load, then listens for change events from any source, reloading configuration and emitting versioned snapshots. Built-in sources do not emit change events yet, but custom sources can opt in via `Watch`.【F:loader.go†L114-L144】【F:types.go†L18-L23】

## Data Models

- **Source / SourceWithKeys**: Contracts for fetching configuration and reporting provenance-friendly key mappings. Sources name themselves (e.g., `env:APP_`, `file:config.yaml`) to appear in provenance and error messages.【F:types.go†L9-L28】
- **ChangeEvent**: Represents a configuration change with timestamp and cause description, emitted by `Source.Watch`. Sources that cannot watch return `ErrWatchNotSupported`.【F:types.go†L29-L36】
- **Loader[T]**: Generic entry point that holds ordered sources, a validator list, and strictness setting. Exposes `Load` for one-time binding/validation and `Watch` for continuous reloads with debouncing handled inside the loop.【F:loader.go†L12-L144】
- **Optional[T]**: Wrapper distinguishing unset values from explicit zero values; provides helpers to access or default the wrapped value.【F:types.go†L37-L48】
- **Validator / ValidatorFunc**: Interfaces for custom validation passes executed after tag-based checks. Functions can be adapted via `ValidatorFunc` to simplify declaration.【F:types.go†L39-L55】
- **Snapshot[T]**: Versioned record produced by `Watch`, containing the loaded config, monotonically increasing version, load timestamp, and the source trigger that initiated the reload.【F:types.go†L56-L63】
- **Provenance / FieldProvenance**: Metadata describing where each field's value originated, including user-visible field path, normalized key path, source identifier, and secret status. Stored alongside configurations for later inspection via `GetProvenance`.【F:provenance.go†L1-L32】
