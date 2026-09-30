package domain_test

import (
	"strings"
	"testing"

	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

func scope() (organisation, year, from, to ids.UUID) {
	return ids.New(), ids.New(), ids.New(), ids.New()
}

// TestAnEdgeNeedsTwoDistinctPeople covers the three structural rules a
// relationship must satisfy before it can exist at all.
func TestAnEdgeNeedsTwoDistinctPeople(t *testing.T) {
	organisation, year, from, to := scope()

	cases := map[string]struct {
		organisation, year, from, to ids.UUID
		relationshipType             domain.RelationshipType
	}{
		"no organisation": {ids.UUID{}, year, from, to, domain.RelationshipGuardianOf},
		"no year":         {organisation, ids.UUID{}, from, to, domain.RelationshipGuardianOf},
		"no people":       {organisation, year, ids.UUID{}, to, domain.RelationshipGuardianOf},
		"self":            {organisation, year, from, from, domain.RelationshipGuardianOf},
		"unknown type":    {organisation, year, from, to, domain.RelationshipType("COUSIN_OF")},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			relationship, err := domain.NewRelationship(
				testCase.organisation, testCase.year, testCase.from, testCase.to, testCase.relationshipType,
			)
			if err == nil {
				t.Fatalf("a relationship was accepted: %+v", relationship)
			}
		})
	}
}

// A symmetric fact must be stored once, or the same relationship would appear
// twice in a family tree with no way to tell which row is the real one.
func TestSymmetricEdgesAreStoredInOneCanonicalDirection(t *testing.T) {
	_, _, low, high := scope()
	// The test cannot assume which identifier sorts lower, so it compares the
	// two constructions rather than against a fixed value.
	first, err := domain.NewRelationship(ids.New(), ids.New(), low, high, domain.RelationshipSiblingOf)
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewRelationship(ids.New(), ids.New(), high, low, domain.RelationshipSiblingOf)
	if err != nil {
		t.Fatal(err)
	}

	if first.FromPersonID != second.FromPersonID || first.ToPersonID != second.ToPersonID {
		t.Errorf("the two directions were not normalised: %s->%s and %s->%s",
			first.FromPersonID, first.ToPersonID, second.FromPersonID, second.ToPersonID)
	}

	// A one-way relationship keeps the direction the caller gave, because
	// "A guardians B" and "B guardians A" mean different things.
	forward, err := domain.NewRelationship(ids.New(), ids.New(), low, high, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}
	backward, err := domain.NewRelationship(ids.New(), ids.New(), high, low, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}
	if forward.FromPersonID != low || backward.FromPersonID != high {
		t.Error("a directed relationship had its direction rewritten")
	}
}

// Only a guardianship can be the billing guardian, so Finance never has to
// interpret "primary" on some other kind of edge.
func TestOnlyAGuardianEdgeCanBePrimary(t *testing.T) {
	organisation, year, from, to := scope()

	guardian, err := domain.NewRelationship(organisation, year, from, to, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}
	if err := guardian.MakePrimary(); err != nil {
		t.Fatalf("a guardian edge cannot be primary: %v", err)
	}
	if !guardian.IsPrimary {
		t.Error("the guardian edge was not marked primary")
	}

	for _, other := range []domain.RelationshipType{
		domain.RelationshipEmergencyContactOf,
		domain.RelationshipSponsorOf,
		domain.RelationshipNextOfKinOf,
		domain.RelationshipSiblingOf,
	} {
		edge, err := domain.NewRelationship(organisation, year, from, to, other)
		if err != nil {
			t.Fatal(err)
		}
		if err := edge.MakePrimary(); err == nil {
			t.Errorf("a %s edge was allowed to be primary", other)
		}
	}
}

// The label vocabulary differs between a French, British and bilingual school, so
// it is free text with a length limit rather than a closed set.
func TestRelationshipRoleIsFreeTextWithinALengthLimit(t *testing.T) {
	organisation, year, from, to := scope()

	edge, err := domain.NewRelationship(organisation, year, from, to, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}

	if err := edge.SetRole("  tutrice  "); err != nil {
		t.Fatal(err)
	}
	if edge.Role != "TUTRICE" {
		t.Errorf("role = %q, want the trimmed upper-cased label", edge.Role)
	}

	if err := edge.SetRole(domain.RelationshipRole(strings.Repeat("x", 65))); err == nil {
		t.Error("a label past the length limit was accepted")
	}
}

// A relationship ends, it is not erased: a statement printed last year must still
// resolve the guardian who was on it then.
func TestDeactivatingARelationshipKeepsTheRow(t *testing.T) {
	organisation, year, from, to := scope()

	edge, err := domain.NewRelationship(organisation, year, from, to, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}
	if !edge.IsActive {
		t.Fatal("a new relationship is not active")
	}

	edge.Deactivate()
	if edge.IsActive {
		t.Error("the relationship is still active after deactivating it")
	}
	if edge.ID.IsNil() {
		t.Error("deactivating cleared the identity, so the edge could not be found again")
	}

	edge.Activate()
	if !edge.IsActive {
		t.Error("the relationship was not reactivated")
	}
}

// The cases that a per-relationship-type schema cannot express without adding a
// table each. They are all just edges here, which is the point of ADR-011.
func TestCrossRoleRelationshipsNeedNoSpecialCase(t *testing.T) {
	organisation, year := ids.New(), ids.New()
	employee, pupil, colleague := ids.New(), ids.New(), ids.New()

	// An employee who is also a guardian: the source holds two roles in the
	// year, and the edge between the two people is unremarkable.
	employeeAsGuardian, err := domain.NewRelationship(organisation, year, employee, pupil, domain.RelationshipGuardianOf)
	if err != nil {
		t.Fatal(err)
	}

	// A guardian who is the emergency contact for a colleague: the target is a
	// member of staff, not a student, and the edge type says so.
	guardianAsContact, err := domain.NewRelationship(organisation, year, employee, colleague, domain.RelationshipEmergencyContactOf)
	if err != nil {
		t.Fatal(err)
	}

	// A guardian with children in several classes is several edges from one
	// source; there is no limit on the fan-out and no separate table.
	second, third := ids.New(), ids.New()
	for _, child := range []ids.UUID{pupil, second, third} {
		if _, err := domain.NewRelationship(organisation, year, employee, child, domain.RelationshipGuardianOf); err != nil {
			t.Errorf("a guardian of several students was rejected: %v", err)
		}
	}

	if employeeAsGuardian.Type != domain.RelationshipGuardianOf {
		t.Error("the employee-as-guardian edge lost its type")
	}
	if guardianAsContact.Type != domain.RelationshipEmergencyContactOf {
		t.Error("the emergency contact edge lost its type")
	}
}

func TestAddMemberStampsTheHouseholdAndNormalisesTheRole(t *testing.T) {
	organisation, year := ids.New(), ids.New()

	household, err := domain.NewHousehold(organisation, year, "Famille Traore")
	if err != nil {
		t.Fatal(err)
	}
	person := ids.New()

	stored, err := household.AddMember(domain.HouseholdMember{PersonID: person})
	if err != nil {
		t.Fatal(err)
	}

	// The caller must be able to persist exactly what the aggregate validated,
	// or the write carries a zero household identifier or a rejected role.
	if stored.HouseholdID != household.ID {
		t.Errorf("stored household = %s, want %s", stored.HouseholdID, household.ID)
	}
	if stored.Role != domain.MemberOther {
		t.Errorf("role = %q, want an empty role normalised to OTHER", stored.Role)
	}
	if len(household.Members) != 1 {
		t.Errorf("member count = %d, want 1", len(household.Members))
	}
}

func TestHouseholdRejectsInvalidMembership(t *testing.T) {
	organisation, year := ids.New(), ids.New()

	household, err := domain.NewHousehold(organisation, year, "Famille Traore")
	if err != nil {
		t.Fatal(err)
	}
	person := ids.New()

	if _, err := household.AddMember(domain.HouseholdMember{PersonID: person, Role: domain.MemberHead}); err != nil {
		t.Fatal(err)
	}
	if _, err := household.AddMember(domain.HouseholdMember{PersonID: person, Role: domain.MemberPartner}); err == nil {
		t.Error("the same person was added to one household twice")
	}
	if _, err := household.AddMember(domain.HouseholdMember{PersonID: ids.UUID{}}); err == nil {
		t.Error("a household accepted a member who is not a person")
	}
	if _, err := household.AddMember(domain.HouseholdMember{
		PersonID: ids.New(),
		Role:     domain.MemberRole("SIBLING"),
	}); err == nil {
		t.Error("a household accepted a role its own database constraint would reject")
	}
}

func TestHouseholdRequiresAName(t *testing.T) {
	organisation, year := ids.New(), ids.New()

	if _, err := domain.NewHousehold(organisation, year, "   "); err == nil {
		t.Error("a household with no billing name was accepted")
	}
	if _, err := domain.NewHousehold(organisation, year, strings.Repeat("x", 201)); err == nil {
		t.Error("a household name past the length limit was accepted")
	}
}
