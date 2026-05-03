package incus

import (
	"testing"
	"time"

	"github.com/lxc/incus/v6/shared/api"
)

func TestToInstance(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	in := &api.Instance{
		InstancePut: api.InstancePut{
			Architecture: "x86_64",
			Config: map[string]string{
				"image.alias":         "images:debian/13",
				"volatile.base_image": "deadbeef",
			},
		},
		Name:      "web-1",
		Type:      "container",
		Status:    "Running",
		CreatedAt: created,
	}
	out := ToInstance(in)
	if out.Name != "web-1" {
		t.Errorf("Name: got %q want web-1", out.Name)
	}
	if out.Type != "container" {
		t.Errorf("Type: got %q want container", out.Type)
	}
	if out.Status != "running" {
		t.Errorf("Status casing: got %q want running", out.Status)
	}
	if out.Image != "images:debian/13" {
		t.Errorf("Image: got %q want alias", out.Image)
	}
	if !out.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt: got %v want %v", out.CreatedAt, created)
	}
}

func TestToInstanceFingerprintFallback(t *testing.T) {
	t.Parallel()
	in := &api.Instance{
		InstancePut: api.InstancePut{
			Config: map[string]string{
				"volatile.base_image": "fp-only",
			},
		},
		Name:   "x",
		Type:   "container",
		Status: "Stopped",
	}
	out := ToInstance(in)
	if out.Image != "fp-only" {
		t.Errorf("fingerprint fallback: got %q want fp-only", out.Image)
	}
	if out.Status != "stopped" {
		t.Errorf("Status: got %q want stopped", out.Status)
	}
}

func TestToInstanceNil(t *testing.T) {
	t.Parallel()
	out := ToInstance(nil)
	if out.Name != "" || out.Status != "" {
		t.Fatalf("nil input: got %+v want zero", out)
	}
}
