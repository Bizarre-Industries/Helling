packer {
  required_plugins {
    parallels = {
      version = "1.2.8"
      source  = "github.com/parallels/parallels"
    }
  }
}

variable "vm_name" {
  type    = string
  default = "helling-dev"
}

variable "debian_arch" {
  type    = string
  default = "arm64"

  validation {
    condition     = contains(["amd64", "arm64"], var.debian_arch)
    error_message = "The debian_arch value must be amd64 or arm64."
  }
}

variable "debian_version" {
  type    = string
  default = "13.4.0"
}

variable "iso_url" {
  type    = string
  default = ""
}

variable "iso_checksum" {
  type    = string
  default = ""
}

variable "cpus" {
  type    = number
  default = 4
}

variable "memory_mb" {
  type    = number
  default = 8192
}

variable "disk_size_mb" {
  type    = number
  default = 40960
}

variable "output_directory" {
  type    = string
  default = "../../../dist/packer/helling-dev"
}

variable "ssh_username" {
  type    = string
  default = "helling"
}

variable "ssh_password_hash" {
  sensitive = true
  type      = string

  validation {
    condition     = var.ssh_password_hash != ""
    error_message = "The ssh_password_hash value must be supplied by scripts/parallels-vm-build-dev.sh."
  }
}

variable "ssh_private_key_file" {
  type = string

  validation {
    condition     = var.ssh_private_key_file != ""
    error_message = "The ssh_private_key_file value must be supplied by scripts/parallels-vm-build-dev.sh."
  }
}

variable "ssh_public_key" {
  type    = string
  default = ""
}

variable "go_version" {
  type    = string
  default = "1.26.2"
}

variable "go_sha256" {
  type    = string
  default = ""
}

variable "startup_view" {
  type    = string
  default = "headless"
}

locals {
  debian_134_checksums = {
    amd64 = "0b813535dd76f2ea96eff908c65e8521512c92a0631fd41c95756ffd7d4896dc"
    arm64 = "c31f8534597df52bd310f716d271bda30a1f58e6ff8fd9e8254eba66776c42d9"
  }
  default_iso_url          = "https://cdimage.debian.org/debian-cd/${var.debian_version}/${var.debian_arch}/iso-cd/debian-${var.debian_version}-${var.debian_arch}-netinst.iso"
  default_iso_checksum     = "sha256:${local.debian_134_checksums[var.debian_arch]}"
  iso_url                  = var.iso_url != "" ? var.iso_url : local.default_iso_url
  iso_checksum             = var.iso_checksum != "" ? var.iso_checksum : local.default_iso_checksum
  parallels_tools_flavor   = var.debian_arch == "arm64" ? "lin-arm" : "lin"
  preseed_installer_locale = "en_US.UTF-8"
  ssh_public_key_b64       = base64encode(var.ssh_public_key)
}

source "parallels-iso" "debian_dev" {
  vm_name          = var.vm_name
  guest_os_type    = "debian"
  startup_view     = var.startup_view
  cpus             = var.cpus
  memory           = var.memory_mb
  disk_size        = var.disk_size_mb
  disk_type        = "expand"
  output_directory = var.output_directory

  iso_url      = local.iso_url
  iso_checksum = local.iso_checksum

  parallels_tools_flavor = local.parallels_tools_flavor
  parallels_tools_mode   = "upload"

  http_network_protocol = "tcp4"
  http_content = {
    "/preseed.cfg" = templatefile("${path.root}/http/preseed.cfg.pkrtpl", {
      hostname        = var.vm_name
      locale          = local.preseed_installer_locale
      ssh_username    = var.ssh_username
      ssh_password_hash = var.ssh_password_hash
      ssh_public_key_b64 = local.ssh_public_key_b64
    })
  }

  boot_wait = "10s"
  boot_command = [
    "<esc><wait>",
    "install ",
    "auto=true priority=critical ",
    "preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
    "debian-installer=${local.preseed_installer_locale} locale=${local.preseed_installer_locale} ",
    "hostname={{ .Name }} fb=false debconf/frontend=noninteractive ",
    "<enter>"
  ]

  ssh_username = var.ssh_username
  ssh_private_key_file = var.ssh_private_key_file
  ssh_timeout  = "45m"

  shutdown_command = "sudo shutdown -P now"
  shutdown_timeout = "10m"

  prlctl = [
    ["set", "{{.Name}}", "--device-set", "net0", "--type", "bridged"],
    ["set", "{{.Name}}", "--on-window-close", "keep-running"]
  ]
}

build {
  name    = "helling-parallels-dev"
  sources = ["source.parallels-iso.debian_dev"]

  provisioner "file" {
    source      = "${path.root}/../../../scripts/install-tools.sh"
    destination = "/tmp/install-tools.sh"
  }

  provisioner "shell" {
    script = "${path.root}/scripts/provision-helling-dev.sh"
    environment_vars = [
      "HELLING_VM_USER=${var.ssh_username}",
      "HELLING_GO_VERSION=${var.go_version}",
      "HELLING_GO_SHA256=${var.go_sha256}",
      "HELLING_DEBIAN_ARCH=${var.debian_arch}",
      "HELLING_PARALLELS_TOOLS_FLAVOR=${local.parallels_tools_flavor}"
    ]
    execute_command = "{{ .Vars }} sudo -E bash '{{ .Path }}'"
  }
}
