# Empty States

> Every page with zero data guides the user forward. Empty states have three parts: what this page is for, a primary action button, and a help link. Never show "No data." alone.

## Ant Design Components

- `antd`: `Empty`, `Button`, `Typography.Text`, `Typography.Link`, `Space`, `Alert`
- `lucide-react`: contextual icons per page

## Design Rules

1. **Every empty state has three parts.** (1) Short explanation of what belongs here. (2) Primary action button to create the first item. (3) Secondary link to docs or alternative action.
2. **Purpose statement first.** Tell the user what this page manages before telling them it's empty. "No virtual machines or system containers yet." not "No data."
3. **Primary action is prominent.** `<Button type="primary">` for the main create action. Secondary actions as `<Button>` or text links.
4. **Help links point to docs.** "New to Helling?" links to quick start guide. Specific pages link to relevant documentation.
5. **Warning tone for safety-critical pages.** `/backups` empty state uses warning icon and urgent language. Your data is not protected.
6. **Recommendations where applicable.** `/firewall` with zero rules warns that all traffic is allowed and recommends deny-all default.
7. **No generic empty illustrations.** No cartoon characters, no "nothing here" SVGs. Text and buttons. Information density.
8. **Templates mention built-ins.** When custom templates are empty, remind users that ~50 built-in templates still exist.
9. **Consistent vertical centering.** Empty state content centers vertically in the content area. Uses antd `<Empty>` with custom `description` and `children`.

## Code Pattern

### Standard Empty State

```tsx
import { Empty, Button, Typography, Space } from "antd";
import { Plus, BookOpen, Upload } from "lucide-react";

const { Text, Link } = Typography;

function EmptyInstances() {
  return (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={
        <Space direction="vertical" size={4}>
          <Text>No virtual machines or system containers yet.</Text>
          <Text type="secondary">
            New to Helling? Create your first virtual machine in 60 seconds.{" "}
            <Link href="/docs/quick-start">Quick Start</Link>
          </Text>
        </Space>
      }
    >
      <Space>
        <Button type="primary" icon={<Plus size={14} />}>
          Create Instance
        </Button>
        <Button icon={<BookOpen size={14} />}>Deploy from Template</Button>
      </Space>
    </Empty>
  );
}
```

### Per-Page Empty States

```tsx
// /instances (zero instances)
<Empty description={
  <Space direction="vertical" size={4}>
    <Text>No virtual machines or system containers yet.</Text>
    <Text type="secondary">
      New to Helling? Create your first virtual machine in 60 seconds.{' '}
      <Link href="/docs/quick-start">Quick Start</Link>
    </Text>
  </Space>
}>
  <Space>
    <Button type="primary">Create Instance</Button>
    <Button>Deploy from Template</Button>
  </Space>
</Empty>

// /containers (zero containers)
<Empty description={
  <Text>No application containers yet.</Text>
}>
  <Space>
    <Button type="primary">Create Container</Button>
    <Button>Deploy from Template</Button>
    <Button icon={<Upload size={14} />}>Import Compose File</Button>
  </Space>
</Empty>

// /kubernetes (zero clusters)
<Empty description={
  <Space direction="vertical" size={4}>
    <Text>No Kubernetes clusters yet.</Text>
    <Text type="secondary">
      Helling creates K8s clusters from Incus VMs with full lifecycle management.
    </Text>
  </Space>
}>
  <Button type="primary">Create Cluster</Button>
</Empty>

// /backups (zero backups) -- warning tone
<Empty
  image={Empty.PRESENTED_IMAGE_SIMPLE}
  description={
    <Space direction="vertical" size={4}>
      <Text type="warning" strong>No backups configured. Your data is not protected.</Text>
      <Text type="secondary">
        Helling can automatically back up all your instances on a schedule.
      </Text>
    </Space>
  }
>
  <Button type="primary">Configure Backup Schedule</Button>
</Empty>

// /firewall (zero rules) -- recommendation
<Empty description={
  <Space direction="vertical" size={4}>
    <Text>No firewall rules. All traffic is allowed to all instances.</Text>
    <Text type="secondary">
      Recommended: start with deny-all and add specific allow rules.
    </Text>
  </Space>
}>
  <Space>
    <Button type="primary">Create Rule</Button>
    <Button>Apply Default Policy</Button>
  </Space>
</Empty>

// /templates (zero custom templates)
<Empty description={
  <Space direction="vertical" size={4}>
    <Text>No custom templates. Helling ships with ~50 built-in app templates.</Text>
    <Text type="secondary">
      You can also <Link>add a custom template repository</Link> or{' '}
      <Link>convert a running instance to a template</Link>.
    </Text>
  </Space>
}>
  <Button type="primary">Browse Built-in Templates</Button>
</Empty>
```

## Standards References

- `docs/design/identity.md` -- per-page empty state text
- `docs/design/identity.md` -- runtime environment (Incus + Podman from first boot)
- `docs/design/philosophy.md` -- Rule 2 (information density), Rule 4 (selectable text)

## Pages Using This Pattern

- `/instances` -- zero VMs/CTs
- `/containers` -- zero app containers
- `/kubernetes` -- zero clusters
- `/backups` -- zero backups (warning tone)
- `/firewall` -- zero rules (recommendation)
- `/templates` -- zero custom templates (mention built-ins)
- `/workspaces` -- zero active workspaces
- `/storage` -- zero storage pools
- `/networking` -- zero networks
- `/images` -- zero local images
- `/bmc` -- zero BMC servers
- `/audit` -- zero audit entries (informational: "No actions recorded yet")
