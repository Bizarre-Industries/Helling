# Images

> Status: Draft

Route: `/images`

> **Data source (ADR-014):** Incus proxy (`/api/incus/1.0/*`). Responses in native Incus format. Podman images via Podman proxy (`/api/podman/libpod/*`).

---

## Layout

Sidebar: resource tree (Images not a top-level item; accessed via sidebar link or global search). Main panel: 3 Tabs.

## API Endpoints

- `GET /api/incus/1.0/images` -- local Incus images
- `GET /api/incus/1.0/images/:fingerprint` -- image detail
- `DELETE /api/incus/1.0/images/:fingerprint` -- delete image
- `POST /api/incus/1.0/images` -- upload/import image
- `GET /api/incus/1.0/images/remotes` -- remote image servers (images.linuxcontainers.org)
- `POST /api/incus/1.0/images` -- pull remote image (with source)
- `GET /api/podman/libpod/images/json` -- Podman images
- `GET /api/incus/1.0/profiles` -- Incus profiles (templates)

## Components

- `Tabs` -- Local Images | Remote Images | Templates

**Local Images tab:** `ProTable` (alias, fingerprint Typography.Text copyable, OS, architecture, type Tag, size, created_at). Actions: Create Instance, Delete. `Upload.Dragger` for ISO/qcow2 upload.

**Remote Images tab:** `ProTable` with search (OS, architecture, type filters via Select). "Download" Button per row to pull to local. Server selector `Select` (default: images.linuxcontainers.org).

**Templates tab:** `ProTable` of Incus profiles marked as templates (name, description, devices, config). "Create Instance from Template" Button.

## Data Model

- Image: `fingerprint`, `aliases[]`, `architecture`, `os`, `type` (container/vm), `size`, `created_at`, `properties{}`
- PodmanImage: `id`, `repository`, `tag`, `size`, `created_at`
- Profile: `name`, `description`, `config{}`, `devices{}`

## States

### Empty State

"No local images. Pull an image from a remote server or upload one." [Browse Remote Images] [Upload Image]

### Loading State

Local images cached. Remote image list fetches on tab switch with inline spinner.

### Error State

Remote server unreachable: inline alert on Remote tab. Local images still shown.

## User Actions

- Upload ISO/qcow2 via drag-and-drop
- Browse and search remote image servers
- Pull remote image to local storage
- Create instance from local image or template
- Delete local images
- Filter by OS, architecture, type

## Cross-References

- Spec: docs/spec/webui-spec.md (Images section)
