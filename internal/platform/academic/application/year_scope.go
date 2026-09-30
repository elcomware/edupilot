package application

import (
	"context"

	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// This file is the Academic module's side of the narrow port that People depends
// on. People must be able to check that an academic year is real and belongs to
// its tenant, but must not gain the ability to create, edit or reorder a school
// calendar as a side effect of registering a student. The two methods below are
// the entire surface exposed for that purpose: *Service therefore satisfies
// people/application.AcademicYearScope without People ever importing Academic's
// repositories, and without Academic importing People.

// BelongsToOrganisation reports whether the academic year exists and belongs to
// the organisation. A year from another tenant is reported as false rather than
// as an error, because from People's side the two cases are the same fact: this
// year is not usable here.
func (s *Service) BelongsToOrganisation(ctx context.Context, organisationID, academicYearID ids.UUID) (bool, error) {
	if organisationID.IsNil() || academicYearID.IsNil() {
		return false, nil
	}

	year, err := s.years.ByID(ctx, organisationID, academicYearID)
	if err != nil {
		return false, err
	}
	return year != nil, nil
}

// CurrentYearID returns the organisation's current academic year, or a nil
// identifier when it has not opened a year yet. A nil identifier is what tells
// People to reject the command rather than guess a year.
func (s *Service) CurrentYearID(ctx context.Context, organisationID ids.UUID) (ids.UUID, error) {
	if organisationID.IsNil() {
		return ids.UUID{}, nil
	}

	year, err := s.years.Current(ctx, organisationID)
	if err != nil {
		return ids.UUID{}, err
	}
	if year == nil {
		return ids.UUID{}, nil
	}
	return year.ID, nil
}
