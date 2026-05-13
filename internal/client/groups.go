package client

import (
	"context"
	"fmt"
)

// Group is a minimal representation suitable for the exporter — just enough
// to identify the group when querying member counts.
type Group struct {
	PK   string
	Name string
}

// ListGroups paginates through /core/groups/ and returns every group. Heavy
// fields (users list, parents, children, inherited roles) are suppressed in
// the request to keep responses small.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var all []Group
	page := int32(1)
	for {
		res, _, err := c.api.CoreApi.CoreGroupsList(c.ctx(ctx)).
			IncludeUsers(false).
			IncludeChildren(false).
			IncludeParents(false).
			IncludeInheritedRoles(false).
			PageSize(100).
			Page(page).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("list groups page %d: %w", page, err)
		}
		for _, g := range res.Results {
			all = append(all, Group{PK: g.Pk, Name: g.Name})
		}
		if int32(res.Pagination.Current) >= int32(res.Pagination.TotalPages) {
			break
		}
		page++
	}
	return all, nil
}
