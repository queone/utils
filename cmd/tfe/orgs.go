// orgs.go

package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/go-tfe"
)

// matches reports whether name contains filter, ignoring case; an empty filter matches everything.
func matches(name, filter string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

// listOrganizations prints every organization whose name matches filter, one per line.
func listOrganizations(ctx context.Context, a api, filter string, w io.Writer) error {
	var all []*tfe.Organization
	for page := 1; page != 0; {
		items, nextPage, err := a.ListOrganizations(ctx, page)
		if err != nil {
			return fmt.Errorf("listing organizations: %w", err)
		}
		all = append(all, items...)
		page = nextPage
	}
	for _, o := range all {
		if matches(o.Name, filter) {
			fmt.Fprintln(w, o.Name)
		}
	}
	return nil
}
