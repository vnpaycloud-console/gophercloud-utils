package helpers

import (
	"context"
	"fmt"

	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/vnpaycloud-console/gophercloud/v2/openstack/networking/v2/ports"
)

type ProjectPurgeOpts struct {
	ComputePurgeOpts *ComputePurgeOpts
	StoragePurgeOpts *StoragePurgeOpts
	NetworkPurgeOpts *NetworkPurgeOpts
}

type ComputePurgeOpts struct {
	// Client is a reference to a specific compute service client.
	Client *gophercloud.ServiceClient
}

type StoragePurgeOpts struct {
	// Client is a reference to a specific storage service client.
	Client *gophercloud.ServiceClient
}

type NetworkPurgeOpts struct {
	// Client is a reference to a specific networking service client.
	Client *gophercloud.ServiceClient
}

// ProjectPurgeAll purges all the resources associated with a project.
// This includes: servers, snapshosts, volumes, floating IPs, routers, networks, sub-networks and security groups
func ProjectPurgeAll(ctx context.Context, projectID string, purgeOpts ProjectPurgeOpts) (err error) {
	if purgeOpts.ComputePurgeOpts != nil {
		err = ProjectPurgeCompute(ctx, projectID, *purgeOpts.ComputePurgeOpts)
		if err != nil {
			return err
		}
	}
	if purgeOpts.StoragePurgeOpts != nil {
		err = ProjectPurgeStorage(ctx, projectID, *purgeOpts.StoragePurgeOpts)
		if err != nil {
			return err
		}
	}
	if purgeOpts.NetworkPurgeOpts != nil {
		err = ProjectPurgeNetwork(ctx, projectID, *purgeOpts.NetworkPurgeOpts)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProjectPurgeCompute purges the Compute v2 resources associated with a project.
// This includes: servers
func ProjectPurgeCompute(ctx context.Context, projectID string, purgeOpts ComputePurgeOpts) (err error) {
	// Delete servers
	listOpts := servers.ListOpts{
		AllTenants: true,
		TenantID:   projectID,
	}

	allPages, err := servers.List(purgeOpts.Client, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding servers for project: " + projectID)
	}

	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting servers for project: " + projectID)
	}

	if len(allServers) > 0 {
		for _, server := range allServers {
			err = servers.Delete(ctx, purgeOpts.Client, server.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting server: " + server.Name + " from project: " + projectID)
			}
		}
	}

	return nil
}

// ProjectPurgeStorage purges the Blockstorage v3 resources associated with a project.
// This includes: snapshosts and volumes
func ProjectPurgeStorage(ctx context.Context, projectID string, purgeOpts StoragePurgeOpts) (err error) {
	// Delete snapshots
	err = clearBlockStorageSnaphosts(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}
	// Delete volumes
	err = clearBlockStorageVolumes(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}

	return nil
}

// ProjectPurgeNetwork purges the Networking v2 resources associated with a project.
// This includes: floating IPs, routers, networks, sub-networks and security groups
func ProjectPurgeNetwork(ctx context.Context, projectID string, purgeOpts NetworkPurgeOpts) (err error) {
	// Delete floating IPs
	err = clearNetworkingFloatingIPs(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}
	// Delete ports
	err = clearNetworkingPorts(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}
	// Delete routers
	err = clearNetworkingRouters(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}
	// Delete networks
	err = clearNetworkingNetworks(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}
	// Delete security groups
	err = clearNetworkingSecurityGroups(ctx, projectID, purgeOpts.Client)
	if err != nil {
		return err
	}

	return nil
}

func clearBlockStorageVolumes(ctx context.Context, projectID string, storageClient *gophercloud.ServiceClient) error {
	listOpts := volumes.ListOpts{
		AllTenants: true,
		TenantID:   projectID,
	}
	allPages, err := volumes.List(storageClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding volumes for project: " + projectID)
	}
	allVolumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting volumes for project: " + projectID)
	}
	if len(allVolumes) > 0 {
		deleteOpts := volumes.DeleteOpts{
			Cascade: true,
		}
		for _, volume := range allVolumes {
			err = volumes.Delete(ctx, storageClient, volume.ID, deleteOpts).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting volume: " + volume.Name + " from project: " + projectID)
			}
		}
	}

	return err
}

func clearBlockStorageSnaphosts(ctx context.Context, projectID string, storageClient *gophercloud.ServiceClient) error {
	listOpts := snapshots.ListOpts{
		AllTenants: true,
		TenantID:   projectID,
	}
	allPages, err := snapshots.List(storageClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding snapshots for project: " + projectID)
	}
	allSnapshots, err := snapshots.ExtractSnapshots(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting snapshots for project: " + projectID)
	}
	if len(allSnapshots) > 0 {
		for _, snaphost := range allSnapshots {
			err = snapshots.Delete(ctx, storageClient, snaphost.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting snaphost: " + snaphost.Name + " from project: " + projectID)
			}
		}
	}
	return nil
}

func clearPortforwarding(ctx context.Context, networkClient *gophercloud.ServiceClient, fipID string, projectID string) error {
	allPages, err := portforwarding.List(networkClient, portforwarding.ListOpts{}, fipID).AllPages(ctx)
	if err != nil {
		return err
	}

	allPFs, err := portforwarding.ExtractPortForwardings(allPages)
	if err != nil {
		return err
	}

	for _, pf := range allPFs {
		err := portforwarding.Delete(ctx, networkClient, fipID, pf.ID).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error deleting floating IP port forwarding: " + pf.ID + " from project: " + projectID)
		}
	}

	return nil
}

func clearNetworkingFloatingIPs(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) error {
	listOpts := floatingips.ListOpts{
		TenantID: projectID,
	}
	allPages, err := floatingips.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding floating IPs for project: " + projectID)
	}
	allFloatings, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting floating IPs for project: " + projectID)
	}
	if len(allFloatings) > 0 {
		for _, floating := range allFloatings {
			// Clear all portforwarding settings otherwise the floating IP can't be deleted
			err = clearPortforwarding(ctx, networkClient, floating.ID, projectID)
			if err != nil {
				return err
			}

			err = floatingips.Delete(ctx, networkClient, floating.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting floating IP: " + floating.ID + " from project: " + projectID)
			}
		}
	}

	return nil
}

func clearNetworkingPorts(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) error {
	listOpts := ports.ListOpts{
		TenantID: projectID,
	}

	allPages, err := ports.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding ports for project: " + projectID)
	}
	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting ports for project: " + projectID)
	}
	if len(allPorts) > 0 {
		for _, port := range allPorts {
			if port.DeviceOwner == "network:ha_router_replicated_interface" {
				continue
			}

			err = ports.Delete(ctx, networkClient, port.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting port: " + port.ID + " from project: " + projectID)
			}
		}
	}

	return nil
}

// We need all subnets to disassociate the router from the subnet
func getAllSubnets(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) ([]string, error) {
	subnets := make([]string, 0)
	listOpts := networks.ListOpts{
		TenantID: projectID,
	}

	allPages, err := networks.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return subnets, fmt.Errorf("Error finding networks for project: " + projectID)
	}
	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return subnets, fmt.Errorf("Error extracting networks for project: " + projectID)
	}
	if len(allNetworks) > 0 {
		for _, network := range allNetworks {
			subnets = append(subnets, network.Subnets...)
		}
	}

	return subnets, nil
}

func clearAllRouterInterfaces(ctx context.Context, projectID string, routerID string, subnets []string, networkClient *gophercloud.ServiceClient) error {
	for _, subnet := range subnets {
		intOpts := routers.RemoveInterfaceOpts{
			SubnetID: subnet,
		}

		_, err := routers.RemoveInterface(ctx, networkClient, routerID, intOpts).Extract()
		if err != nil {
			return err
		}
	}

	return nil
}

func clearNetworkingRouters(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) error {
	listOpts := routers.ListOpts{
		TenantID: projectID,
	}
	allPages, err := routers.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding routers for project: " + projectID)
	}
	allRouters, err := routers.ExtractRouters(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting routers for project: " + projectID)
	}

	subnets, err := getAllSubnets(ctx, projectID, networkClient)
	if err != nil {
		return fmt.Errorf("Error fetching subnets project: " + projectID)
	}

	if len(allRouters) > 0 {
		for _, router := range allRouters {
			err = clearAllRouterInterfaces(ctx, projectID, router.ID, subnets, networkClient)
			if err != nil {
				return err
			}

			routes := []routers.Route{}
			// Clear all routes
			updateOpts := routers.UpdateOpts{
				Routes: &routes,
			}

			_, err := routers.Update(ctx, networkClient, router.ID, updateOpts).Extract()
			if err != nil {
				return fmt.Errorf("Error deleting router: " + router.Name + " from project: " + projectID)
			}

			err = routers.Delete(ctx, networkClient, router.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting router: " + router.Name + " from project: " + projectID)
			}
		}
	}

	return nil
}

func clearNetworkingNetworks(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) error {
	listOpts := networks.ListOpts{
		TenantID: projectID,
	}

	allPages, err := networks.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding networks for project: " + projectID)
	}
	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting networks for project: " + projectID)
	}
	if len(allNetworks) > 0 {
		for _, network := range allNetworks {
			err = networks.Delete(ctx, networkClient, network.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting network: " + network.Name + " from project: " + projectID)
			}
		}
	}

	return nil
}

func clearNetworkingSecurityGroups(ctx context.Context, projectID string, networkClient *gophercloud.ServiceClient) error {
	listOpts := groups.ListOpts{
		TenantID: projectID,
	}
	allPages, err := groups.List(networkClient, listOpts).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("Error finding security groups for project: " + projectID)
	}
	allSecGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return fmt.Errorf("Error extracting security groups for project: " + projectID)
	}
	if len(allSecGroups) > 0 {
		for _, group := range allSecGroups {
			err = groups.Delete(ctx, networkClient, group.ID).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting security group: " + group.Name + " from project: " + projectID)
			}
		}
	}

	return nil
}
