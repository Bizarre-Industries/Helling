package incus

import (
	"strings"
	"time"

	"github.com/lxc/incus/v6/shared/api"
)

// Instance is the API-shaped instance record returned from /v1/instances.
// Mirrors components.schemas.Instance in api/openapi.yaml.
type Instance struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Image        string    `json:"image,omitempty"`
	Architecture string    `json:"architecture,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	IPv4         []string  `json:"ipv4,omitempty"`
	IPv6         []string  `json:"ipv6,omitempty"`
}

// ToInstance converts an Incus api.Instance into our shape. Status casing is
// normalised to lowercase to match the OpenAPI enum.
func ToInstance(in *api.Instance) Instance {
	if in == nil {
		return Instance{}
	}
	out := Instance{
		Name:         in.Name,
		Type:         in.Type,
		Status:       strings.ToLower(in.Status),
		Architecture: in.Architecture,
		CreatedAt:    in.CreatedAt,
	}
	if alias, ok := in.Config["image.alias"]; ok {
		out.Image = alias
	} else if fp, ok := in.Config["volatile.base_image"]; ok {
		out.Image = fp
	}
	return out
}
