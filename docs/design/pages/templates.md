# Templates

> Status: Draft

Route: `/templates`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

Sidebar: "Templates" selected. Main panel: category filter bar + card grid. This is one of the few pages that uses cards (app template gallery is an explicit exception to the tables-by-default rule).

## API Endpoints

- `GET /api/v1/templates` -- list built-in + custom templates
- `GET /api/v1/templates/:id` -- template detail
- `POST /api/v1/templates/:id/deploy` -- deploy from template
- `GET /api/v1/templates/repos` -- custom template repos
- `POST /api/v1/templates/repos` -- add custom repo

## Components

- `ProList` with `grid={{ column: 4 }}` -- template card grid
- `Card` -- cover image (logo), title, description, category `Tag`, "Deploy" `Button` (primary)
- `Input.Search` -- search/filter templates by name
- `Segmented` or `Tag` group -- category filter (Media, Dev Tools, Monitoring, Networking, Databases, etc.)
- `ModalForm` -- deploy form with customizable env vars (name, port, data path). Advanced toggle reveals Monaco YAML editor.
- `Tabs` -- App Templates | Workspace Templates

## Data Model

- Template: `id`, `title`, `description`, `logo`, `category`, `env_vars[]`, `compose_yaml`, `type` (container/instance/workspace)
- EnvVar: `key`, `label`, `default`, `required`, `description`
- Repo: `url`, `name`, `last_synced`

## States

### Empty State
"No custom templates. Helling ships with ~50 built-in app templates." "You can also [add a custom template repository] or [convert a running instance to a template]."

### Loading State
Show cached template list immediately. Background refresh.

### Error State
If custom repo unreachable: inline warning on that repo card. Built-in templates always available.

## User Actions

- Browse and search templates by name/category
- Deploy template via ModalForm (3 fields for simple, YAML editor for advanced)
- Add/remove custom template repository URL
- Convert running instance to custom template

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Templates section)
- Identity: docs/design/identity.md (First Action -- template deploy flow)
