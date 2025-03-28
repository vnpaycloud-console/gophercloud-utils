package testing

import (
	"testing"

	"github.com/vnpaycloud-console/gophercloud-utils/v2/openstack/objectstorage/v1/objects"
	th "github.com/vnpaycloud-console/gophercloud/v2/testhelper"
)

func TestContainerPartition(t *testing.T) {
	containerName := "foo/bar/baz"

	expectedContainerName := "foo"
	expectedPseudoFolder := "bar/baz"

	actualContainerName, actualPseudoFolder := objects.ContainerPartition(containerName)
	th.AssertEquals(t, expectedContainerName, actualContainerName)
	th.AssertEquals(t, expectedPseudoFolder, actualPseudoFolder)

	containerName = "foo"
	expectedContainerName = "foo"
	expectedPseudoFolder = ""

	actualContainerName, actualPseudoFolder = objects.ContainerPartition(containerName)
	th.AssertEquals(t, expectedContainerName, actualContainerName)
	th.AssertEquals(t, expectedPseudoFolder, actualPseudoFolder)
}
