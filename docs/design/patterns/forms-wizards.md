# Forms and Wizards

> Every create/edit operation in Helling uses ProForm or StepsForm. Progressive disclosure hides 80+ advanced knobs behind a toggle so new users see 5 fields and power users see everything. Every field has a tooltip.

## Ant Design Components

- `@ant-design/pro-components`: `StepsForm`, `ProForm`, `ProFormText`, `ProFormSelect`, `ProFormSlider`, `ProFormSwitch`, `ProFormDigit`, `ProFormTextArea`, `ProFormDependency`, `ModalForm`, `DrawerForm`
- `antd`: `Tooltip`, `Typography.Text`, `Alert`, `Segmented`, `Collapse`
- `@uiw/react-codemirror`: raw YAML/config editing (dynamic import)

## Design Rules

1. **StepsForm for create wizards.** VMs, containers, K8s clusters, storage pools, blueprints. Multi-step with validation per step. Step state persists across navigation.
2. **ModalForm for quick adds.** Snapshots, firewall rules, backup schedules, tags. Single-step, opens as modal over current page.
3. **Progressive disclosure.** Default mode shows essential fields (name, CPU, RAM, disk, network). `<Segmented>` toggle between "Simple" and "Advanced" reveals all hardware knobs. Never overwhelm new users.
4. **Tooltip on every field.** Explain what it does, when to change it, what the default is, performance implications. Use `tooltip` prop on ProForm fields.
5. **Conditional fields.** Use `<ProFormDependency>` to show/hide fields based on other values. Example: Secure Boot toggle only visible when BIOS type is UEFI. Balloon minimum only visible when ballooning is enabled.
6. **Validation inline.** Error messages appear under the field, not in a summary. Use `rules` prop on every field. Required fields marked with asterisk.
7. **No Zod, no separate form library.** ProForm handles validation natively. Keep it simple.
8. **Review step.** Every StepsForm ends with a Review step showing all selections as `<Descriptions>`. Users confirm before submit.
9. **Default values are safe.** CPU type: `host`. Disk cache: `none`. NIC model: `virtio`. Boot firmware: OVMF. Every default explained in tooltip.
10. **CodeMirror for raw config.** Cloud-init YAML, compose YAML, hookscripts, raw QEMU overrides. Dynamic import only when needed.

## Code Pattern

### StepsForm (VM Create Wizard with Progressive Disclosure)

```tsx
import {
  StepsForm,
  ProFormText,
  ProFormSelect,
  ProFormSlider,
  ProFormSwitch,
  ProFormDigit,
  ProFormDependency,
} from "@ant-design/pro-components";
import { Descriptions, Segmented, Collapse, Alert } from "antd";
import { useState } from "react";

export function CreateInstanceWizard({
  onFinish,
}: {
  onFinish: (values: any) => Promise<void>;
}) {
  const [mode, setMode] = useState<"simple" | "advanced">("simple");

  return (
    <StepsForm onFinish={onFinish}>
      {/* Step 1: Basics */}
      <StepsForm.StepForm name="basics" title="Basics">
        <ProFormText
          name="name"
          label="Name"
          rules={[
            { required: true },
            {
              pattern: /^[a-z0-9-]+$/,
              message: "Lowercase, numbers, hyphens only",
            },
          ]}
          tooltip="Instance hostname. Used for DNS: name.helling.local"
        />
        <ProFormSelect
          name="image"
          label="OS Image"
          rules={[{ required: true }]}
          tooltip="Operating system to install. Ubuntu recommended for first-time users."
          showSearch
          request={fetchImages}
        />
        <Segmented
          options={["Simple", "Advanced"]}
          value={mode === "simple" ? "Simple" : "Advanced"}
          onChange={(v) => setMode(v === "Simple" ? "simple" : "advanced")}
          style={{ marginBottom: 16 }}
        />
      </StepsForm.StepForm>

      {/* Step 2: Resources */}
      <StepsForm.StepForm name="resources" title="Resources">
        <ProFormSlider
          name="cpu"
          label="CPU Cores"
          min={1}
          max={64}
          initialValue={2}
          tooltip="Number of virtual CPU cores. Use 'host' CPU type for best performance."
        />
        <ProFormSlider
          name="memory"
          label="RAM (GB)"
          min={0.5}
          max={256}
          step={0.5}
          initialValue={4}
          tooltip="Leave at least 1GB for the host. Windows needs 4GB minimum."
        />
        <ProFormSlider
          name="disk"
          label="Disk (GB)"
          min={5}
          max={2000}
          step={5}
          initialValue={30}
          tooltip="Root disk size. Can be grown later but not shrunk."
        />

        {mode === "advanced" && (
          <Collapse
            ghost
            items={[
              {
                key: "cpu-advanced",
                label: "CPU Advanced",
                children: (
                  <>
                    <ProFormSelect
                      name="cpuType"
                      label="CPU Type"
                      initialValue="host"
                      tooltip="Use 'host' for best performance. Use named types for live migration."
                      options={[
                        { label: "host (best performance)", value: "host" },
                        { label: "kvm64 (safe default)", value: "kvm64" },
                        { label: "max (all host features)", value: "max" },
                        { label: "EPYC-Genoa", value: "EPYC-Genoa" },
                        { label: "Skylake-Server", value: "Skylake-Server" },
                      ]}
                    />
                    <ProFormSwitch
                      name="numa"
                      label="NUMA"
                      tooltip="Enable on multi-socket hosts for optimal memory locality."
                    />
                    <ProFormText
                      name="cpuAffinity"
                      label="CPU Pinning"
                      tooltip="Pin to specific cores (e.g., 0-3 or 0,2,4,6). Use with NUMA."
                    />
                  </>
                ),
              },
              {
                key: "memory-advanced",
                label: "Memory Advanced",
                children: (
                  <>
                    <ProFormSwitch
                      name="balloon"
                      label="Ballooning"
                      tooltip="Reclaims unused RAM for other VMs. Slight performance cost."
                    />
                    <ProFormDependency name={["balloon"]}>
                      {({ balloon }) =>
                        balloon && (
                          <ProFormSlider
                            name="balloonMin"
                            label="Minimum RAM (GB)"
                            min={0.5}
                            max={256}
                            step={0.5}
                          />
                        )
                      }
                    </ProFormDependency>
                    <ProFormSelect
                      name="hugepages"
                      label="Hugepages"
                      initialValue="none"
                      tooltip="1GB hugepages = lowest latency. Requires host hugepage reservation."
                      options={[
                        { label: "None", value: "none" },
                        { label: "2MB", value: "2MB" },
                        { label: "1GB", value: "1GB" },
                      ]}
                    />
                  </>
                ),
              },
              {
                key: "system-advanced",
                label: "System",
                children: (
                  <>
                    <ProFormSelect
                      name="firmware"
                      label="Firmware"
                      initialValue="OVMF"
                      tooltip="OVMF (UEFI) recommended. SeaBIOS for legacy OS."
                      options={[
                        { label: "OVMF (UEFI)", value: "OVMF" },
                        { label: "SeaBIOS (Legacy)", value: "SeaBIOS" },
                      ]}
                    />
                    <ProFormDependency name={["firmware"]}>
                      {({ firmware }) =>
                        firmware === "OVMF" && (
                          <ProFormSwitch
                            name="secureBoot"
                            label="Secure Boot"
                            tooltip="Requires UEFI firmware. Recommended for Windows 11."
                          />
                        )
                      }
                    </ProFormDependency>
                    <ProFormSelect
                      name="tpm"
                      label="TPM"
                      initialValue="none"
                      tooltip="TPM v2.0 required for Windows 11."
                      options={[
                        { label: "None", value: "none" },
                        { label: "v2.0 (recommended)", value: "2.0" },
                        { label: "v1.2 (legacy)", value: "1.2" },
                      ]}
                    />
                    <ProFormSwitch
                      name="guestAgent"
                      label="QEMU Guest Agent"
                      initialValue={true}
                      tooltip="Enables freeze-on-backup and filesystem info."
                    />
                  </>
                ),
              },
            ]}
          />
        )}
      </StepsForm.StepForm>

      {/* Step 3: Network */}
      <StepsForm.StepForm name="network" title="Network">
        <ProFormSelect
          name="network"
          label="Network"
          rules={[{ required: true }]}
          tooltip="Bridge or OVN network. Instances on the same network can communicate."
          request={fetchNetworks}
        />
        {mode === "advanced" && (
          <>
            <ProFormSelect
              name="nicModel"
              label="NIC Model"
              initialValue="virtio"
              options={[
                { label: "VirtIO (best)", value: "virtio" },
                { label: "e1000 (compatible)", value: "e1000" },
              ]}
            />
            <ProFormDigit
              name="vlan"
              label="VLAN Tag"
              tooltip="Leave empty for untagged traffic."
            />
          </>
        )}
      </StepsForm.StepForm>

      {/* Step 4: Cloud-Init */}
      <StepsForm.StepForm name="cloudinit" title="Cloud-Init">
        <ProFormText
          name="user"
          label="Username"
          initialValue="admin"
          tooltip="Default login username."
        />
        <ProFormTextArea
          name="sshKeys"
          label="SSH Public Keys"
          tooltip="One key per line. Paste from GitHub: curl https://github.com/USER.keys"
          rows={3}
        />
      </StepsForm.StepForm>

      {/* Step 5: Review */}
      <StepsForm.StepForm name="review" title="Review">
        <Alert
          message="An automatic snapshot will be created before any future config changes."
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        {/* Populated by StepsForm values via ProFormDependency */}
      </StepsForm.StepForm>
    </StepsForm>
  );
}
```

### ModalForm (Quick Add)

```tsx
import {
  ModalForm,
  ProFormText,
  ProFormSelect,
} from "@ant-design/pro-components";
import { Button } from "antd";

export function AddFirewallRuleButton({
  instanceName,
}: {
  instanceName: string;
}) {
  return (
    <ModalForm
      title="Add Firewall Rule"
      trigger={<Button type="primary">Add Rule</Button>}
      onFinish={async (values) => {
        await createFirewallRule(instanceName, values);
        return true; // closes modal
      }}
    >
      <ProFormSelect
        name="action"
        label="Action"
        rules={[{ required: true }]}
        options={[
          { label: "Accept", value: "accept" },
          { label: "Drop", value: "drop" },
          { label: "Reject", value: "reject" },
        ]}
      />
      <ProFormSelect
        name="direction"
        label="Direction"
        rules={[{ required: true }]}
        options={[
          { label: "Inbound", value: "in" },
          { label: "Outbound", value: "out" },
        ]}
      />
      <ProFormSelect
        name="protocol"
        label="Protocol"
        options={[
          { label: "TCP", value: "tcp" },
          { label: "UDP", value: "udp" },
          { label: "ICMP", value: "icmp" },
        ]}
      />
      <ProFormText
        name="source"
        label="Source"
        tooltip="CIDR notation, e.g., 192.168.1.0/24. Leave empty for any."
      />
      <ProFormText
        name="dport"
        label="Destination Port"
        tooltip="Single port (22), range (8000-9000), or comma-separated (80,443)."
      />
    </ModalForm>
  );
}
```

## Standards References

- `docs/design/philosophy.md` -- Rule 2 (information density), Rule 9 (no framework bloat)
- `docs/spec/compute.md` -- Part 5: progressive disclosure pattern, simple vs advanced mode
- `docs/spec/webui-spec.md` -- StepsForm for create wizards, ModalForm for quick adds
- `CLAUDE.md` -- ProForm/StepsForm replaces custom forms, no Zod

## Pages Using This Pattern

- `/instances` -- Create Instance wizard (StepsForm, 5-10 steps depending on mode)
- `/containers` -- Create Container (ModalForm), Deploy from Compose (ModalForm with CodeMirror)
- `/kubernetes` -- Create Cluster wizard (StepsForm, 6 steps)
- `/storage` -- Create Pool (StepsForm), Volume operations (ModalForm)
- `/networking` -- Create Network (ModalForm)
- `/firewall` -- Add Rule (ModalForm), Create Security Group (ModalForm)
- `/backups` -- Configure Schedule (ModalForm), Backup Now (ModalForm)
- `/templates` -- Deploy Template (ModalForm with env vars)
- `/users` -- Create User (ModalForm), 2FA Setup (StepsForm)
- `/settings` -- each tab uses ProForm for settings persistence
- `/bmc` -- Add Server (ModalForm)
