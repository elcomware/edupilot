package main

import (
	"context"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	academicapp "github.com/elcomware/edupilot/internal/platform/academic/application"
	peopleapp "github.com/elcomware/edupilot/internal/platform/people/application"
	peopledomain "github.com/elcomware/edupilot/internal/platform/people/domain"
)

// The App service itself is declared in app_service.go. This file holds the
// academic and people operations, which are the year-scoped half of the
// interface: nearly every one of them is about "as they stand in this academic
// year", which is what the People module is for.

// tenant resolves the installation's organisation, returning the failure envelope
// the transport uses for every error.
func (a *App) tenant(ctx context.Context) (ids.UUID, error) {
	organisation, err := a.organisations.GetCurrentOrganisation(ctx)
	if err != nil {
		return ids.UUID{}, err
	}
	return organisation.ID, nil
}

// ListYears returns the academic years of the current organisation. It backs
// platform.academic.App.ListYears.
func (a *App) ListYears(ctx context.Context, request api.PageRequest) api.Result[api.Page[api.AcademicYearView]] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.Page[api.AcademicYearView]](err)
	}

	years, err := a.academic.ListYears(ctx, organisationID)
	if err != nil {
		return api.Fail[api.Page[api.AcademicYearView]](err)
	}
	return api.OK(api.NewPage(api.NewAcademicYearViews(years), request))
}

// GetCurrentYear returns the organisation's open academic year. It backs
// platform.academic.App.GetCurrentYear.
//
// A school that has not opened a year yet gets a NOT_FOUND failure rather than an
// empty success, so the interface can say what to do about it.
func (a *App) GetCurrentYear(ctx context.Context) api.Result[api.AcademicYearView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}

	year, err := a.academic.GetCurrentYear(ctx, organisationID)
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}
	return api.OK(api.NewAcademicYearView(year))
}

// CreateYear opens an academic year. It backs platform.academic.App.CreateYear.
func (a *App) CreateYear(ctx context.Context, request api.CreateYearRequest) api.Result[api.AcademicYearView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}

	startsOn, err := api.ParseRequiredCalendarDate("startsOn", request.StartsOn)
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}
	endsOn, err := api.ParseRequiredCalendarDate("endsOn", request.EndsOn)
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}

	year, err := a.academic.CreateYear(ctx, academicapp.CreateYearCommand{
		OrganisationID: organisationID,
		Name:           request.Name,
		StartsOn:       startsOn,
		EndsOn:         endsOn,
		MakeCurrent:    request.MakeCurrent,
	})
	if err != nil {
		return api.Fail[api.AcademicYearView](err)
	}
	return api.OK(api.NewAcademicYearView(year))
}

// ListTerms returns the terms of one academic year. It backs
// platform.academic.App.ListTerms.
func (a *App) ListTerms(ctx context.Context, request api.ListTermsRequest) api.Result[[]api.TermView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[[]api.TermView](err)
	}
	yearID, ok := api.ParseID(request.AcademicYearID)
	if !ok {
		return api.Fail[[]api.TermView](api.InvalidID("academicYearId", request.AcademicYearID))
	}

	terms, err := a.academic.ListTerms(ctx, organisationID, yearID)
	if err != nil {
		return api.Fail[[]api.TermView](err)
	}
	return api.OK(api.NewTermViews(terms))
}

// ListPeople returns a page of people. It backs platform.people.App.List.
func (a *App) ListPeople(ctx context.Context, request api.ListPeopleRequest) api.Result[api.Page[api.PersonView]] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}

	people, err := a.people.ListPeople(ctx, organisationID, request.Search, request.IncludeInactive)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}

	// The rows carry role badges, so the roles are read once for the whole list
	// rather than once per row.
	personIDs := make([]ids.UUID, 0, len(people))
	for _, person := range people {
		personIDs = append(personIDs, person.ID)
	}
	rolesByPerson, err := a.people.ListRolesForPeople(ctx, organisationID, personIDs, yearID)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}

	views := make([]api.PersonView, 0, len(people))
	for _, person := range people {
		views = append(views, api.NewPersonViewWithRoles(person, rolesByPerson[person.ID]))
	}
	return api.OK(api.NewPage(views, request.PageRequest))
}

// GetPerson returns one person. It backs platform.people.App.Get.
func (a *App) GetPerson(ctx context.Context, request api.GetPersonRequest) api.Result[api.PersonView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.PersonView](err)
	}
	personID, ok := api.ParseID(request.ID)
	if !ok {
		return api.Fail[api.PersonView](api.InvalidID("id", request.ID))
	}

	person, err := a.people.GetPerson(ctx, organisationID, personID)
	if err != nil {
		return api.Fail[api.PersonView](err)
	}

	// The roles are resolved here because the interface shows a person as they
	// stand in the current year, and the current year is the one the school is
	// working in unless it says otherwise.
	roles, err := a.people.ListRoleDetails(ctx, organisationID, personID, ids.UUID{})
	if err != nil {
		return api.Fail[api.PersonView](err)
	}
	return api.OK(api.NewPersonViewWithRoles(person, roles))
}

// RegisterPerson adds a person. It backs platform.people.App.Register.
func (a *App) RegisterPerson(ctx context.Context, request api.RegisterPersonRequest) api.Result[api.PersonView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.PersonView](err)
	}
	dateOfBirth, err := api.ParseCalendarDate(request.DateOfBirth)
	if err != nil {
		return api.Fail[api.PersonView](err)
	}

	person, err := a.people.RegisterPerson(ctx, peopleapp.RegisterPersonCommand{
		OrganisationID: organisationID,
		FirstName:      request.FirstName,
		MiddleName:     request.MiddleName,
		LastName:       request.LastName,
		PreferredName:  request.PreferredName,
		Email:          request.Email,
		Phone:          request.Phone,
		SecondaryPhone: request.SecondaryPhone,
		DateOfBirth:    dateOfBirth,
		Gender:         request.Gender,
		Nationality:    request.Nationality,
		NationalID:     request.NationalID,
		Address: peopledomain.Address{
			Line1:      request.Address.Line1,
			Line2:      request.Address.Line2,
			City:       request.Address.City,
			PostalCode: request.Address.PostalCode,
			Country:    request.Address.Country,
		},
	})
	if err != nil {
		return api.Fail[api.PersonView](err)
	}
	return api.OK(api.NewPersonView(person, nil))
}

// AssignRole gives a person a role in an academic year. It backs
// platform.people.App.AssignRole.
func (a *App) AssignRole(ctx context.Context, request api.AssignRoleRequest) api.Result[api.PersonRoleView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	personID, ok := api.ParseID(request.PersonID)
	if !ok {
		return api.Fail[api.PersonRoleView](api.InvalidID("personId", request.PersonID))
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	startsOn, err := api.ParseCalendarDate(request.StartsOn)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	endsOn, err := api.ParseCalendarDate(request.EndsOn)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	admissionDate, err := api.ParseCalendarDate(request.AdmissionDate)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	hiredOn, err := api.ParseCalendarDate(request.HiredOn)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	endedOn, err := api.ParseCalendarDate(request.EndedOn)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}

	role, err := a.people.AssignRole(ctx, peopleapp.AssignRoleCommand{
		OrganisationID: organisationID,
		PersonID:       personID,
		AcademicYearID: yearID,
		Role:           peopledomain.Role(request.Role),
		StartsOn:       startsOn,
		EndsOn:         endsOn,
		StudentNumber:  request.StudentNumber,
		AdmissionDate:  admissionDate,
		StudentStatus:  peopledomain.StudentStatus(request.StudentStatus),
		PreviousSchool: request.PreviousSchool,
		IsBoarding:     request.IsBoarding,
		EmployeeNumber: request.EmployeeNumber,
		JobTitle:       request.JobTitle,
		Department:     request.Department,
		HiredOn:        hiredOn,
		EndedOn:        endedOn,
		ContractType:   peopledomain.ContractType(request.ContractType),
		PayrollGroup:   request.PayrollGroup,
		// Rights are passed as a block rather than four loose flags so that
		// "no guardian rights at all" stays distinguishable from "an emergency
		// contact who may not collect the child".
		GuardianRights: &peopleapp.GuardianRights{
			IsEmergencyContact:  request.IsEmergencyContact,
			MayCollectStudent:   request.MayCollectStudent,
			IsBillingContact:    request.IsBillingContact,
			CanAuthoriseMedical: request.CanAuthoriseMedical,
		},
	})
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}

	// Re-read through GetRole so the caller gets the profile that was just
	// written, rather than a role with no student number beside it.
	details, err := a.people.GetRole(ctx, organisationID, role.ID)
	if err != nil {
		return api.Fail[api.PersonRoleView](err)
	}
	return api.OK(api.NewPersonRoleView(details.Role, details.Student, details.Employee, details.Guardian))
}

// Link records a relationship between two people. It backs
// platform.people.App.Link.
func (a *App) Link(ctx context.Context, request api.LinkRelationshipRequest) api.Result[api.RelationshipView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	fromID, ok := api.ParseID(request.FromPersonID)
	if !ok {
		return api.Fail[api.RelationshipView](api.InvalidID("fromPersonId", request.FromPersonID))
	}
	toID, ok := api.ParseID(request.ToPersonID)
	if !ok {
		return api.Fail[api.RelationshipView](api.InvalidID("toPersonId", request.ToPersonID))
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}

	relationship, err := a.people.Link(ctx, peopleapp.LinkRelationshipCommand{
		OrganisationID: organisationID,
		AcademicYearID: yearID,
		FromPersonID:   fromID,
		ToPersonID:     toID,
		Type:           peopledomain.RelationshipType(request.Type),
		Role:           peopledomain.RelationshipRole(request.Role),
		MakePrimary:    request.MakePrimary,
	})
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	return api.OK(api.NewRelationshipView(relationship))
}

// Unlink ends a relationship without deleting it. It backs
// platform.people.App.Unlink.
func (a *App) Unlink(ctx context.Context, request api.UnlinkRelationshipRequest) api.Result[api.RelationshipView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	relationshipID, ok := api.ParseID(request.ID)
	if !ok {
		return api.Fail[api.RelationshipView](api.InvalidID("id", request.ID))
	}

	if err := a.people.Unlink(ctx, organisationID, relationshipID); err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	return a.relationship(ctx, api.GetRelationshipRequest{ID: request.ID})
}

// relationship returns one relationship. It is deliberately unexported: the
// interface has no screen for reading a single edge, and a bound method is part
// of the transport's surface whether or not anybody needs it.
func (a *App) relationship(ctx context.Context, request api.GetRelationshipRequest) api.Result[api.RelationshipView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	relationshipID, ok := api.ParseID(request.ID)
	if !ok {
		return api.Fail[api.RelationshipView](api.InvalidID("id", request.ID))
	}

	relationship, err := a.people.GetRelationship(ctx, organisationID, relationshipID)
	if err != nil {
		return api.Fail[api.RelationshipView](err)
	}
	return api.OK(api.NewRelationshipView(relationship))
}

// ListRelationships returns the edges around one person in a year. It backs
// platform.people.App.Relationships.
func (a *App) ListRelationships(ctx context.Context, request api.ListRelationshipsRequest) api.Result[[]api.RelationshipView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[[]api.RelationshipView](err)
	}
	personID, ok := api.ParseID(request.PersonID)
	if !ok {
		return api.Fail[[]api.RelationshipView](api.InvalidID("personId", request.PersonID))
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[[]api.RelationshipView](err)
	}

	relationshipType := peopledomain.RelationshipType(request.Type)
	if request.Incoming {
		relationships, err := a.people.RelationshipsTo(ctx, organisationID, personID, yearID, relationshipType)
		if err != nil {
			return api.Fail[[]api.RelationshipView](err)
		}
		return api.OK(api.NewRelationshipViews(relationships))
	}

	relationships, err := a.people.RelationshipsOf(ctx, organisationID, personID, yearID, relationshipType)
	if err != nil {
		return api.Fail[[]api.RelationshipView](err)
	}
	return api.OK(api.NewRelationshipViews(relationships))
}

// ListHouseholds returns the billing parties of a year. It backs
// platform.people.App.ListHouseholds.
func (a *App) ListHouseholds(ctx context.Context, request api.ListHouseholdsRequest) api.Result[[]api.HouseholdView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[[]api.HouseholdView](err)
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[[]api.HouseholdView](err)
	}

	households, err := a.people.ListHouseholds(ctx, organisationID, yearID)
	if err != nil {
		return api.Fail[[]api.HouseholdView](err)
	}
	return api.OK(api.NewHouseholdViews(households))
}

// ListPeopleWithRole returns a roster: everyone holding one role in a year. It
// backs platform.people.App.ListPeopleWithRole, which is what the students,
// employees and guardians screens are built on rather than filtering a page of
// everybody client-side.
func (a *App) ListPeopleWithRole(ctx context.Context, request api.ListPeopleWithRoleRequest) api.Result[api.Page[api.PersonView]] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}
	role, err := peopledomain.ParseRole(request.Role)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}

	roles, err := a.people.ListPeopleWithRole(ctx, organisationID, yearID, role)
	if err != nil {
		return api.Fail[api.Page[api.PersonView]](err)
	}

	// A roster is a list of people, so the roles are resolved for each of them
	// and the page is cut here rather than in the interface.
	views := make([]api.PersonView, 0, len(roles))
	for _, held := range roles {
		person, err := a.people.GetPerson(ctx, organisationID, held.PersonID)
		if err != nil {
			return api.Fail[api.Page[api.PersonView]](err)
		}
		details, err := a.people.GetRole(ctx, organisationID, held.ID)
		if err != nil {
			return api.Fail[api.Page[api.PersonView]](err)
		}
		views = append(views, api.NewPersonViewWithRole(person, details))
	}
	return api.OK(api.NewPage(views, api.PageRequest{}))
}

// CreateHousehold creates a billing party. It backs
// platform.people.App.CreateHousehold.
func (a *App) CreateHousehold(ctx context.Context, request api.CreateHouseholdRequest) api.Result[api.HouseholdView] {
	organisationID, err := a.tenant(ctx)
	if err != nil {
		return api.Fail[api.HouseholdView](err)
	}
	yearID, err := api.ParseOptionalID("academicYearId", request.AcademicYearID)
	if err != nil {
		return api.Fail[api.HouseholdView](err)
	}

	members := make([]peopleapp.HouseholdMemberCommand, 0, len(request.Members))
	for _, requested := range request.Members {
		personID, ok := api.ParseID(requested.PersonID)
		if !ok {
			return api.Fail[api.HouseholdView](api.InvalidID("members.personId", requested.PersonID))
		}
		members = append(members, peopleapp.HouseholdMemberCommand{
			PersonID:      personID,
			Role:          peopledomain.MemberRole(requested.Role),
			IsBillingPart: requested.IsBillingPart,
		})
	}

	household, err := a.people.CreateHousehold(ctx, peopleapp.CreateHouseholdCommand{
		OrganisationID: organisationID,
		AcademicYearID: yearID,
		Name:           request.Name,
		BillingEmail:   request.BillingEmail,
		BillingPhone:   request.BillingPhone,
		BillingAddress: peopledomain.Address{
			Line1:      request.BillingAddress.Line1,
			Line2:      request.BillingAddress.Line2,
			City:       request.BillingAddress.City,
			PostalCode: request.BillingAddress.PostalCode,
			Country:    request.BillingAddress.Country,
		},
		Members: members,
	})
	if err != nil {
		return api.Fail[api.HouseholdView](err)
	}
	return api.OK(api.NewHouseholdView(household))
}
