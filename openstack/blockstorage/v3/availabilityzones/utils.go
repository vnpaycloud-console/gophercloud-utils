package availabilityzones

import (
	"context"

	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/blockstorage/v3/availabilityzones"
)

// ListAvailableAvailabilityZones is a convenience function that return a slice of available Availability Zones.
func ListAvailableAvailabilityZones(ctx context.Context, client *gophercloud.ServiceClient) ([]string, error) {
	var zones []string

	allPages, err := availabilityzones.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	availabilityZoneInfo, err := availabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		return nil, err
	}

	// This should always return at at least two AZs. By default, Nova will
	// return an AZ for internal services (typically called 'internal') and AZ
	// for (typically called 'nova'). We can obviously configure additional AZs
	// and you can also configure the names of these default AZs with
	// '[DEFAULT] internal_service_availability_zone' and
	// '[DEFAULT] default_availability_zone', respectively.
	for _, zone := range availabilityZoneInfo {
		if zone.ZoneState.Available {
			zones = append(zones, zone.ZoneName)
		}
	}

	return zones, nil
}
