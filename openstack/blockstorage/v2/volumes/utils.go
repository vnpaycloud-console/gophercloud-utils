package volumes

import (
	"context"

	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/blockstorage/v2/volumes"
)

// IDFromName is a convenience function that returns a volume's ID given its name.
func IDFromName(ctx context.Context, client *gophercloud.ServiceClient, name string) (string, error) {
	count := 0
	id := ""

	listOpts := volumes.ListOpts{
		Name: name,
	}

	pages, err := volumes.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return "", err
	}

	all, err := volumes.ExtractVolumes(pages)
	if err != nil {
		return "", err
	}

	for _, s := range all {
		if s.Name == name {
			count++
			id = s.ID
		}
	}

	switch count {
	case 0:
		return "", gophercloud.ErrResourceNotFound{Name: name, ResourceType: "volume"}
	case 1:
		return id, nil
	default:
		return "", gophercloud.ErrMultipleResourcesFound{Name: name, Count: count, ResourceType: "volume"}
	}
}
