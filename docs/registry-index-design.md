# gpm Registry and Index Design

Status: Draft 0.1
Date: 2026-05-03

## 1. Summary

This document defines the recommended architecture for the future `gpm` registry and its index.

The core decision is:

- proxy-compatible module artifacts are one layer
- canonical package metadata is another layer
- search and discovery indexes are derived data, not the source of truth

`gpm` should not treat "the index" as the registry's canonical state.

## 2. Why This Shape

Go already has strong conventions around module identity and distribution:

- module paths are canonical identities
- Go proxy endpoints define standard fetch behavior
- `go.sum` and checksum infrastructure already anchor integrity

Because of that, the registry should not behave like a giant mutable package document store. That model fits npm more naturally than it fits Go.

The better model for `gpm` is:

1. artifact store for proxy-compatible module payloads
2. immutable version metadata store
3. rebuildable derived indexes for search and discovery

## 3. Goals

The registry/index system should:

1. Stay compatible with the Go proxy protocol.
2. Allow `gpm` to add richer package metadata and search.
3. Keep installs working even if search/index systems are degraded.
4. Make published version records append-only wherever possible.
5. Make search and listing indexes fully rebuildable from canonical data.
6. Support public and private registries.

## 4. Non-Goals

The registry/index system will not:

1. Replace Go module identity rules.
2. Make search/index documents the authoritative package record.
3. Require `gpm add` or `gpm install` to depend on full-text search availability.
4. Introduce a new binary packaging format for Go apps.

## 5. System Layers

### 5.1 Artifact Layer

This layer stores or proxies the Go-compatible module artifacts:

- `.mod`
- `.info`
- `.zip`
- checksum and integrity data

This layer must remain compatible with proxy-style module fetching.

### 5.2 Canonical Metadata Layer

This layer stores the source-of-truth package metadata that `gpm` adds beyond raw proxy artifacts.

Examples:

- module path
- version
- publish time
- visibility
- repository URL
- description
- license
- tags
- deprecation state
- yank or hide state
- organization ownership
- declared CLI binaries if relevant

This layer should be append-only for published versions, with limited soft-state updates for fields such as deprecation or visibility.

### 5.3 Derived Index Layer

This layer exists for search, listing, and discovery.

Examples:

- full-text search
- package latest-version lookup
- org or scope listings
- tag/category listings
- "contains binary command X" lookup
- recently published packages

This layer is disposable and rebuildable.

## 6. Canonical Rule

Installs and resolution must never depend on the derived index.

That means:

- `gpm add github.com/acme/lib@v1.2.3` should work from canonical metadata plus proxy-compatible artifacts
- `gpm install` should work from canonical metadata plus proxy-compatible artifacts
- `gpm search` and registry browsing may degrade if the index is stale or unavailable

If search is down, install should still work.

## 7. Data Model

### 7.1 Package Record

Logical key:

- `module_path`

Suggested fields:

- `module_path`
- `latest_version`
- `latest_stable_version`
- `visibility`
- `repository_url`
- `homepage_url`
- `description`
- `license`
- `owner_type`
- `owner_id`
- `created_at`
- `updated_at`

### 7.2 Version Record

Logical key:

- `(module_path, version)`

Suggested fields:

- `module_path`
- `version`
- `published_at`
- `artifact_source`
- `mod_checksum`
- `zip_checksum`
- `go_version`
- `retracted`
- `deprecated`
- `deprecation_message`
- `hidden`
- `readme_digest`
- `manifest_digest`
- `declared_binaries`

### 7.3 Derived Index Records

Examples:

- search document per package
- latest-version table
- owner-package table
- tag-package table
- binary-name lookup table

None of these are canonical.

## 8. Recommended Storage Model

Use separate storage concerns:

1. artifact/object store for proxy payloads
2. SQL metadata store for package and version truth
3. search index for full-text and discovery

Recommended first implementation:

- artifact layer: filesystem or object storage
- canonical metadata: PostgreSQL or SQLite during early development
- search: database-backed FTS first, dedicated search engine only if needed later

The main goal is to preserve a hard distinction between truth and index, not to over-engineer the first backend.

## 9. Publish Flow

Publishing should write canonical state first, then derive index state.

### 9.1 Publish Steps

1. Validate the package and version.
2. Build or collect proxy-compatible artifacts.
3. Store artifacts.
4. Write canonical package and version metadata transactionally.
5. Emit an indexing event.
6. Update derived indexes asynchronously.

### 9.2 Important Rule

A publish is successful when artifacts and canonical metadata are committed.

Search/index updates are follow-on work, not the commit point of publication.

## 10. Index Update Flow

The index should be fed by canonical metadata changes, not by ad hoc crawling of search documents.

Recommended model:

1. canonical metadata write occurs
2. change event is emitted
3. index worker consumes event
4. derived search/listing tables are updated

If indexing fails:

- package resolution still works
- search may be stale
- the system can retry or rebuild later

## 11. Rebuild Strategy

Every derived index must be fully rebuildable from:

- canonical package metadata
- canonical version metadata
- stored artifacts or artifact descriptors

That gives us:

- disaster recovery
- safer schema evolution
- simpler operational debugging

Rebuild command examples for the future:

```text
gpm registry reindex
gpm registry reindex --package github.com/acme/lib
gpm registry reindex --from-event 12034
```

## 12. API Shape

The future registry likely needs two API families.

### 12.1 Resolution APIs

These serve installation and module-fetching flows.

They must support:

- proxy-compatible resolution
- artifact retrieval
- metadata needed for secure installs

### 12.2 Discovery APIs

These serve search and package browsing flows.

They may support:

- search by text
- list by owner
- list by tag
- binary command discovery
- recent and popular views

Discovery APIs can be stale briefly. Resolution APIs should not depend on discovery freshness.

## 13. CLI Implications

The CLI should reflect the split:

- `gpm add` and `gpm install` use resolution flows
- `gpm search` uses discovery flows
- `gpm publish` writes canonical metadata and artifacts
- `gpm registry reindex` is an operator workflow, not a normal developer path

This keeps the user model honest.

## 14. Failure Modes

### 14.1 Search Index Down

Expected behavior:

- `gpm search` fails or degrades
- direct install by path and version still works

### 14.2 Artifact Store Unavailable

Expected behavior:

- installs fail for affected packages
- search may still work

### 14.3 Metadata Store Unavailable

Expected behavior:

- publish and resolution fail
- search may serve stale data if separately cached

### 14.4 Stale Derived Index

Expected behavior:

- package browse/search results may lag
- direct installs remain correct if canonical metadata and artifacts are available

## 15. Security and Policy

Because canonical metadata is separate from derived index state, policy can attach to package/version truth instead of to search documents.

Examples:

- org visibility rules
- package deprecation
- yanked versions
- vulnerability flags
- license policy status

These flags may then flow into derived indexes and CLI UX, but they should originate in canonical metadata.

## 16. Phase Plan

### Phase 1: CLI Only

- no `gpm` registry implementation yet
- use configured upstream proxies
- no local notion of a canonical registry index

### Phase 2: Minimal Registry

- proxy-compatible artifact service
- canonical metadata store
- basic latest-version and owner/package indexes

### Phase 3: Rich Discovery

- search
- tags/categories
- binary-command discovery
- policy-enriched package views

## 17. Recommendation

Proceed with this rule set:

1. artifact storage is canonical for proxy payloads
2. SQL-backed package/version metadata is canonical for `gpm` metadata
3. search and listing indexes are derived
4. installs must not depend on search/index availability

This is the safest way to add npm-like discovery and publishing UX without compromising Go compatibility or operational clarity.

For the concrete canonical table shapes and API metadata records, see [docs/registry-metadata-schema.md](registry-metadata-schema.md).
