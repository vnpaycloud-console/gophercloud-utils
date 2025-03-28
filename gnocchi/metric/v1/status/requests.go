package status

import (
	"context"

	"github.com/vnpaycloud-console/gophercloud/v2"
)

// GetOptsBuilder allows to add additional parameters to the Get request.
type GetOptsBuilder interface {
	ToStatusGetQuery() (string, error)
}

// GetOpts allows to provide additional options to the Gnocchi status Get request.
type GetOpts struct {
	// Details allows to get status with all attributes.
	Details *bool `q:"details"`
}

// ToStatusGetQuery formats a GetOpts into a query string.
func (opts GetOpts) ToStatusGetQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

// Get retrieves the overall status of the Gnocchi installation.
func Get(ctx context.Context, c *gophercloud.ServiceClient, opts GetOptsBuilder) (r GetResult) {
	url := getURL(c)
	if opts != nil {
		query, err := opts.ToStatusGetQuery()
		if err != nil {
			r.Err = err
			return
		}
		url += query
	}
	resp, err := c.Get(ctx, url, &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
