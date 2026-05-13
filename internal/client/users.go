package client

import (
	"context"
	"fmt"
	"time"
)

// UserCountFilter narrows the user-count query. All fields are optional.
type UserCountFilter struct {
	IsActive     *bool
	DateJoinedGt *time.Time
}

// UserCount returns the total number of users matching the filter, using
// page_size=1 and reading the pagination.count field — a single cheap call
// regardless of total user count.
//
// Note: the authentik API exposes DateJoinedGt (exclusive) but not Gte.
// For "users created in the last N seconds" semantics this is close enough;
// the off-by-one at the second boundary is negligible.
func (c *Client) UserCount(ctx context.Context, f UserCountFilter) (int, error) {
	req := c.api.CoreApi.CoreUsersList(c.ctx(ctx)).PageSize(1)
	if f.IsActive != nil {
		req = req.IsActive(*f.IsActive)
	}
	if f.DateJoinedGt != nil {
		req = req.DateJoinedGt(*f.DateJoinedGt)
	}
	res, _, err := req.Execute()
	if err != nil {
		return 0, fmt.Errorf("users list: %w", err)
	}
	return int(res.Pagination.Count), nil
}
