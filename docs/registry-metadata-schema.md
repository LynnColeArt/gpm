# gpm Registry Metadata Schema

Status: Draft 0.1
Date: 2026-05-03

## 1. Summary

This document defines the canonical metadata schema for the future `gpm` registry.

It is downstream of the registry/index design in [docs/registry-index-design.md](registry-index-design.md) and makes the central split concrete:

- artifact storage is canonical for proxy-compatible payloads
- metadata storage is canonical for package/version facts
- search/listing indexes are derived from canonical metadata

## 2. Design Rules

The schema must follow these rules:

1. Published version records are append-only except for soft state such as deprecation or visibility.
2. Package and version truth lives in canonical metadata tables, not in search documents.
3. Search/listing state must be rebuildable from canonical metadata plus artifact descriptors.
4. Install and resolution flows must not require the search index.

## 3. Canonical Entities

The recommended canonical entities are:

1. `packages`
2. `package_versions`
3. `package_version_artifacts`
4. `package_version_binaries`
5. `package_events`

Everything else should be treated as derived or operational support.

## 4. Packages Table

Logical key:

- `module_path`

Purpose:

- package-level metadata that is stable across versions
- package ownership and visibility
- latest-version pointers for fast resolution

Suggested shape:

```sql
create table packages (
  module_path text primary key,
  visibility text not null,
  owner_type text not null,
  owner_id text not null,
  repository_url text,
  homepage_url text,
  description text,
  license_spdx text,
  latest_version text,
  latest_stable_version text,
  deprecated boolean not null default false,
  deprecation_message text,
  hidden boolean not null default false,
  created_at timestamptz not null,
  updated_at timestamptz not null
);
```

Notes:

- `latest_version` and `latest_stable_version` are canonical convenience pointers, not search index state.
- `description` at the package level is the currently preferred package description, not a replacement for version snapshots.

## 5. Package Versions Table

Logical key:

- `(module_path, version)`

Purpose:

- immutable version-level metadata
- publish-time facts
- policy flags attached to a specific version

Suggested shape:

```sql
create table package_versions (
  module_path text not null,
  version text not null,
  published_at timestamptz not null,
  publish_source text not null,
  go_version text,
  visibility text not null,
  retracted boolean not null default false,
  retract_reason text,
  deprecated boolean not null default false,
  deprecation_message text,
  hidden boolean not null default false,
  readme_digest text,
  manifest_digest text,
  metadata_json jsonb not null default '{}'::jsonb,
  primary key (module_path, version),
  foreign key (module_path) references packages (module_path)
);
```

Notes:

- `publish_source` may be values like `upload`, `mirror`, or `vcs-import`.
- `metadata_json` is an escape hatch for early evolution, not a substitute for normalizing known fields.

## 6. Package Version Artifacts Table

Purpose:

- artifact descriptors for proxy-compatible assets
- checksums and storage references
- provenance about where the fetchable bytes live

Suggested shape:

```sql
create table package_version_artifacts (
  module_path text not null,
  version text not null,
  mod_ref text not null,
  info_ref text not null,
  zip_ref text not null,
  mod_sha256 text,
  info_sha256 text,
  zip_sha256 text,
  source_proxy text,
  source_vcs_url text,
  created_at timestamptz not null,
  primary key (module_path, version),
  foreign key (module_path, version)
    references package_versions (module_path, version)
);
```

Notes:

- refs can point at object storage keys, filesystem paths, or opaque artifact handles.
- canonical metadata should store descriptors, not duplicate artifact bytes.

## 7. Package Version Binaries Table

Purpose:

- declare CLI binaries produced by a specific version
- support future app discovery without making search canonical

Suggested shape:

```sql
create table package_version_binaries (
  module_path text not null,
  version text not null,
  binary_name text not null,
  package_path text not null,
  is_default boolean not null default false,
  primary key (module_path, version, binary_name),
  foreign key (module_path, version)
    references package_versions (module_path, version)
);
```

Notes:

- `binary_name` is the exposed installed command name.
- `package_path` is the Go package path that should be built for that binary.

## 8. Package Events Table

Purpose:

- outbox/change-log stream for index rebuilds and async projections

Suggested shape:

```sql
create table package_events (
  event_id bigint generated always as identity primary key,
  event_type text not null,
  module_path text not null,
  version text,
  payload_json jsonb not null,
  created_at timestamptz not null,
  processed_at timestamptz
);
```

Event examples:

- `package.created`
- `package.updated`
- `package.version.published`
- `package.version.deprecated`
- `package.version.hidden`

## 9. Derived Index Tables

These are intentionally not canonical.

Recommended derived projections:

### 9.1 `package_search_documents`

Purpose:

- full-text search and ranking

Suggested fields:

- `module_path`
- `document_text`
- `tags_json`
- `binary_names_json`
- `updated_at`

### 9.2 `package_owner_listing`

Purpose:

- list packages by owner/org/scope

Suggested fields:

- `owner_type`
- `owner_id`
- `module_path`
- `latest_version`
- `updated_at`

### 9.3 `package_binary_lookup`

Purpose:

- discover packages by CLI binary name

Suggested fields:

- `binary_name`
- `module_path`
- `version`
- `package_path`

These tables should be rebuildable from canonical package/version metadata.

## 10. Publish Transaction Boundary

The canonical publish transaction should commit:

1. package row upsert
2. package version insert
3. package artifact descriptor insert
4. package binary declaration inserts
5. package event append

Index updates should happen after that transaction, not inside it.

## 11. API Serialization Shape

The registry will likely need a stable metadata API representation in addition to proxy endpoints.

Suggested package response:

```json
{
  "modulePath": "github.com/acme/payments",
  "visibility": "private",
  "owner": {
    "type": "org",
    "id": "acme"
  },
  "repositoryUrl": "https://github.com/acme/payments",
  "description": "Payments service and shared tooling",
  "license": "MIT",
  "latestVersion": "v1.4.2",
  "latestStableVersion": "v1.4.2",
  "deprecated": false,
  "hidden": false
}
```

Suggested version response:

```json
{
  "modulePath": "github.com/acme/payments",
  "version": "v1.4.2",
  "publishedAt": "2026-05-03T00:00:00Z",
  "goVersion": "1.25.4",
  "deprecated": false,
  "hidden": false,
  "artifacts": {
    "modRef": "objects/mod/...",
    "infoRef": "objects/info/...",
    "zipRef": "objects/zip/..."
  },
  "binaries": [
    {
      "name": "paymentsctl",
      "packagePath": "github.com/acme/payments/cmd/paymentsctl"
    }
  ]
}
```

## 12. Compatibility Notes

This schema is deliberately designed so that:

- the Go proxy artifact layer remains standard
- the metadata layer can grow independently
- the search index can be rebuilt or replaced later

That gives `gpm` room to add npm-like discovery and publishing UX without compromising the install path.

## 13. Recommendation

When the registry service is implemented:

1. start with canonical package/version/artifact tables plus event outbox
2. keep derived search/listing projections separate
3. do not collapse package truth into one giant mutable search document

That is the cleanest path for long-term operability and Go compatibility.
