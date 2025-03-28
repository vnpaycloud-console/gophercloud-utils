package snapshots

import (
	"context"

	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/blockstorage/v2/snapshots"
)

// IDFromName is a convenience function that returns a snapshot's ID given its name.
func IDFromName(ctx context.Context, client *gophercloud.ServiceClient, name string) (string, error) {
	count := 0
	id := ""

	listOpts := snapshots.ListOpts{
		Name: name,
	}

	pages, err := snapshots.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return "", err
	}

	all, err := snapshots.ExtractSnapshots(pages)
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
		return "", gophercloud.ErrResourceNotFound{Name: name, ResourceType: "snapshot"}
	case 1:
		return id, nil
	default:
		return "", gophercloud.ErrMultipleResourcesFound{Name: name, Count: count, ResourceType: "snapshot"}
	}
}
