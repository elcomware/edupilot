package main

import (
	"context"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/app"
	academicapp "github.com/elcomware/edupilot/internal/platform/academic/application"
	campusapp "github.com/elcomware/edupilot/internal/platform/campus/application"
	organisationapp "github.com/elcomware/edupilot/internal/platform/organisation/application"
	peopleapp "github.com/elcomware/edupilot/internal/platform/people/application"
)

// App is the Wails transport service (Mode A). It adapts the shared
// application services to the frontend gateway contract and holds no business
// rules of its own: cmd/server exposes exactly the same operations over HTTP.
//
// The organisation is resolved from the installation on every call rather than
// read from the request. A single-tenant desktop build has one tenant, and taking
// it from the client would let a caller name somebody else's school.
type App struct {
	organisations *organisationapp.Service
	campuses      *campusapp.Service
	academic      *academicapp.Service
	people        *peopleapp.Service
}

// newApp builds the transport service from the composition root.
func newApp(application *app.Application) *App {
	return &App{
		organisations: application.Organisations,
		campuses:      application.Campuses,
		academic:      application.Academic,
		people:        application.People,
	}
}

// GetCurrent returns the tenant of this installation. It backs
// platform.organisation.App.GetCurrent.
func (a *App) GetCurrent(ctx context.Context) api.Result[api.OrganisationView] {
	organisation, err := a.organisations.GetCurrentOrganisation(ctx)
	if err != nil {
		return api.Fail[api.OrganisationView](err)
	}
	return api.OK(api.NewOrganisationView(organisation))
}

// ListCampuses returns the campuses of the current organisation. It backs
// platform.campus.App.List.
func (a *App) ListCampuses(ctx context.Context, request api.PageRequest) api.Result[api.Page[api.CampusView]] {
	organisation, err := a.organisations.GetCurrentOrganisation(ctx)
	if err != nil {
		return api.Fail[api.Page[api.CampusView]](err)
	}

	campuses, err := a.campuses.ListCampuses(ctx, organisation.ID, true)
	if err != nil {
		return api.Fail[api.Page[api.CampusView]](err)
	}

	return api.OK(api.NewPage(api.NewCampusViews(campuses), request))
}
