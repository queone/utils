// api.go

package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe"
)

// pageSize is the number of items requested per API page.
const pageSize = 100

// api is the narrow slice of the Terraform Cloud API the utility uses. The
// real adapter wraps go-tfe; tests supply a fake. Each list call takes a page
// number and returns the next page, 0 when there is none, so the operations
// themselves walk the pages.
type api interface {
	ListOrganizations(ctx context.Context, page int) ([]*tfe.Organization, int, error)
	ListModules(ctx context.Context, org string, page int) ([]*tfe.RegistryModule, int, error)
	ReadModuleVersion(ctx context.Context, id tfe.RegistryModuleID, version string) (*tfe.RegistryModuleVersion, error)
	ListWorkspaces(ctx context.Context, org string, page int) ([]*tfe.Workspace, int, error)
	ReadWorkspace(ctx context.Context, org, name string) (*tfe.Workspace, error)
	CreateWorkspace(ctx context.Context, org string, options tfe.WorkspaceCreateOptions) (*tfe.Workspace, error)
	ListVariables(ctx context.Context, workspaceID string, page int) ([]*tfe.Variable, int, error)
	CreateVariable(ctx context.Context, workspaceID string, options tfe.VariableCreateOptions) (*tfe.Variable, error)
	ReadAgentPool(ctx context.Context, id string) (*tfe.AgentPool, error)
}

// client adapts a go-tfe client to the api interface.
type client struct {
	c *tfe.Client
}

// newClient builds the real adapter for the given address and token.
func newClient(domain, token string) (api, error) {
	c, err := tfe.NewClient(&tfe.Config{Address: domain, Token: token})
	if err != nil {
		return nil, fmt.Errorf("creating the Terraform Cloud client for %s: %w", domain, err)
	}
	return &client{c: c}, nil
}

// listOptions builds the paging options for one page.
func listOptions(page int) tfe.ListOptions {
	return tfe.ListOptions{PageNumber: page, PageSize: pageSize}
}

// next returns the following page number from a pagination block, 0 at the end.
func next(p *tfe.Pagination) int {
	if p == nil {
		return 0
	}
	return p.NextPage
}

func (a *client) ListOrganizations(ctx context.Context, page int) ([]*tfe.Organization, int, error) {
	l, err := a.c.Organizations.List(ctx, &tfe.OrganizationListOptions{ListOptions: listOptions(page)})
	if err != nil {
		return nil, 0, err
	}
	return l.Items, next(l.Pagination), nil
}

func (a *client) ListModules(ctx context.Context, org string, page int) ([]*tfe.RegistryModule, int, error) {
	l, err := a.c.RegistryModules.List(ctx, org, &tfe.RegistryModuleListOptions{ListOptions: listOptions(page)})
	if err != nil {
		return nil, 0, err
	}
	return l.Items, next(l.Pagination), nil
}

func (a *client) ReadModuleVersion(ctx context.Context, id tfe.RegistryModuleID, version string) (*tfe.RegistryModuleVersion, error) {
	return a.c.RegistryModules.ReadVersion(ctx, id, version)
}

func (a *client) ListWorkspaces(ctx context.Context, org string, page int) ([]*tfe.Workspace, int, error) {
	l, err := a.c.Workspaces.List(ctx, org, &tfe.WorkspaceListOptions{ListOptions: listOptions(page)})
	if err != nil {
		return nil, 0, err
	}
	return l.Items, next(l.Pagination), nil
}

func (a *client) ReadWorkspace(ctx context.Context, org, name string) (*tfe.Workspace, error) {
	return a.c.Workspaces.Read(ctx, org, name)
}

func (a *client) CreateWorkspace(ctx context.Context, org string, options tfe.WorkspaceCreateOptions) (*tfe.Workspace, error) {
	return a.c.Workspaces.Create(ctx, org, options)
}

func (a *client) ListVariables(ctx context.Context, workspaceID string, page int) ([]*tfe.Variable, int, error) {
	l, err := a.c.Variables.List(ctx, workspaceID, &tfe.VariableListOptions{ListOptions: listOptions(page)})
	if err != nil {
		return nil, 0, err
	}
	return l.Items, next(l.Pagination), nil
}

func (a *client) CreateVariable(ctx context.Context, workspaceID string, options tfe.VariableCreateOptions) (*tfe.Variable, error) {
	return a.c.Variables.Create(ctx, workspaceID, options)
}

func (a *client) ReadAgentPool(ctx context.Context, id string) (*tfe.AgentPool, error) {
	return a.c.AgentPools.Read(ctx, id)
}
