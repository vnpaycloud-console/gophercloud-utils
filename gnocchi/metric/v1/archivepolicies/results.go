package archivepolicies

import (
	"github.com/vnpaycloud-console/gophercloud/v2"
	"github.com/vnpaycloud-console/gophercloud/v2/pagination"
)

type commonResult struct {
	gophercloud.Result
}

// Extract is a function that accepts a result and extracts a Gnocchi archive policy.
func (r commonResult) Extract() (*ArchivePolicy, error) {
	var s *ArchivePolicy
	err := r.ExtractInto(&s)
	return s, err
}

// GetResult represents the result of a get operation. Call its Extract
// method to interpret it as an ArchivePolicy.
type GetResult struct {
	commonResult
}

// CreateResult represents the result of a create operation. Call its Extract
// method to interpret it as a Gnocchi archive policy.
type CreateResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation. Call its Extract
// method to interpret it as a Gnocchi archive policy.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation. Call its
// ExtractErr method to determine if the request succeeded or failed.
type DeleteResult struct {
	gophercloud.ErrResult
}

// ArchivePolicy represents a Gnocchi archive policy.
// Archive policy is an aggregate storage policy attached to a metric.
// It determines how long aggregates will be kept in a metric and how they will be aggregated.
type ArchivePolicy struct {
	// AggregationMethods is a list of functions used to aggregate
	// multiple measures into an aggregate.
	AggregationMethods []string `json:"aggregation_methods"`

	// BackWindow configures number of coarsest periods to keep.
	// It allows to process measures that are older
	// than the last timestamp period boundary.
	BackWindow int `json:"back_window"`

	// Definition is a list of parameters that configures
	// archive policy precision and timespan.
	Definition []ArchivePolicyDefinition `json:"definition"`

	// Name is a name of an archive policy.
	Name string `json:"name"`
}

// ArchivePolicyDefinition represents definition of how metrics will
// be saved with the selected archive policy.
// It configures precision and timespan.
type ArchivePolicyDefinition struct {
	// Granularity is the level of  precision that must be kept when aggregating data.
	Granularity string `json:"granularity"`

	// Points is a given aggregates or samples in the lifespan of a time series.
	// Time series is a list of aggregates ordered by time.
	Points int `json:"points"`

	// TimeSpan is the time period for which a metric keeps its aggregates.
	TimeSpan string `json:"timespan"`
}

// ArchivePolicyPage abstracts the raw results of making a List() request against
// the Gnocchi API.
//
// As Gnocchi API may freely alter the response bodies of structures
// returned to the client, you may only safely access the data provided through
// the ExtractArchivePolicies call.
type ArchivePolicyPage struct {
	pagination.SinglePageBase
}

// IsEmpty returns true if an ArchivePolicyPage contains no archive policies.
func (r ArchivePolicyPage) IsEmpty() (bool, error) {
	archivePolicies, err := ExtractArchivePolicies(r)
	return len(archivePolicies) == 0, err
}

// ExtractArchivePolicies interprets the results of a single page from a List() call,
// producing a slice of ArchivePolicy structs.
func ExtractArchivePolicies(r pagination.Page) ([]ArchivePolicy, error) {
	var s []ArchivePolicy
	err := (r.(ArchivePolicyPage)).ExtractInto(&s)
	if err != nil {
		return nil, err
	}

	return s, err
}
