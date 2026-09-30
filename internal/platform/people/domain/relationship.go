package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// RelationshipType is the meaning of a directed edge between two people.
//
// One table carries every relationship in the school, so no domain owns a family
// tree. The three cases that a per-relationship-type schema cannot express
// without multiplying tables all fall out of this list:
//
//	an employee who is also a guardian   Person --GUARDIAN_OF--> Student
//	a guardian who is an emergency contact
//	                                      for a colleague      Person --EMERGENCY_CONTACT_OF--> Person
//	a guardian with children in many classes
//	                                      Person --GUARDIAN_OF--> Student (x N)
type RelationshipType string

// The relationship types EduPilot Core recognises.
const (
	// RelationshipGuardianOf means the source is a guardian of the target, who is
	// normally a student.
	RelationshipGuardianOf RelationshipType = "GUARDIAN_OF"
	// RelationshipEmergencyContactOf means the source may be contacted in an
	// emergency about the target, who may be a student or a member of staff.
	RelationshipEmergencyContactOf RelationshipType = "EMERGENCY_CONTACT_OF"
	// RelationshipSponsorOf means the source sponsors the target's fees.
	RelationshipSponsorOf RelationshipType = "SPONSOR_OF"
	// RelationshipCaregiverOf means the source cares for the target day to day.
	RelationshipCaregiverOf RelationshipType = "CAREGIVER_OF"
	// RelationshipNextOfKinOf means the source is the target's next of kin.
	RelationshipNextOfKinOf RelationshipType = "NEXT_OF_KIN_OF"
	// RelationshipSiblingOf is symmetric and is stored in one canonical direction.
	RelationshipSiblingOf RelationshipType = "SIBLING_OF"
	// RelationshipSpouseOf is symmetric and is stored in one canonical direction.
	RelationshipSpouseOf RelationshipType = "SPOUSE_OF"
	// RelationshipStepParentOf means the source is a step-parent of the target.
	RelationshipStepParentOf RelationshipType = "STEP_PARENT_OF"
)

// Symmetric reports whether a relationship points both ways, so that the same
// fact is never stored twice. A symmetric edge is stored with the lower person
// identifier as the source, which makes the unique index do the deduplication.
func (t RelationshipType) Symmetric() bool {
	switch t {
	case RelationshipSiblingOf, RelationshipSpouseOf:
		return true
	default:
		return false
	}
}

// Valid reports whether the relationship type is one EduPilot knows.
func (t RelationshipType) Valid() bool {
	switch t {
	case RelationshipGuardianOf, RelationshipEmergencyContactOf, RelationshipSponsorOf,
		RelationshipCaregiverOf, RelationshipNextOfKinOf, RelationshipSiblingOf,
		RelationshipSpouseOf, RelationshipStepParentOf:
		return true
	default:
		return false
	}
}

// RelationshipRole is the specific label on an edge, such as MOTHER or FATHER for
// a guardian, or SIBLING for an emergency contact. It is free text because the
// vocabulary differs between a French, British and bilingual school, and a
// school may need a label EduPilot has never heard of.
type RelationshipRole string

// Common relationship roles. They are suggestions for the interface, not a
// closed set: any non-empty label is accepted.
const (
	RelationshipMother        RelationshipRole = "MOTHER"
	RelationshipFather        RelationshipRole = "FATHER"
	RelationshipParent        RelationshipRole = "PARENT"
	RelationshipLegalGuardian RelationshipRole = "LEGAL_GUARDIAN"
	RelationshipFosterParent  RelationshipRole = "FOSTER_PARENT"
	RelationshipGrandparent   RelationshipRole = "GRANDPARENT"
	RelationshipSibling       RelationshipRole = "SIBLING"
	RelationshipAunt          RelationshipRole = "AUNT"
	RelationshipUncle         RelationshipRole = "UNCLE"
	RelationshipSpouse        RelationshipRole = "SPOUSE"
	RelationshipEmployer      RelationshipRole = "EMPLOYER"
	RelationshipFriend        RelationshipRole = "FRIEND"
	RelationshipNanny         RelationshipRole = "NANNY"
)

const maxRelationshipRoleLength = 64

// Relationship is one directed, typed edge between two people, scoped to an
// academic year.
//
// The year scope matters: a guardianship ends. A year-less edge would either have
// to be deleted, losing history, or would need a manual end date that disagrees
// with the school calendar.
type Relationship struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	FromPersonID   ids.UUID
	ToPersonID     ids.UUID
	Type           RelationshipType
	Role           RelationshipRole
	IsPrimary      bool
	IsActive       bool
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewRelationship creates a relationship with a generated identifier.
//
// It normalises a symmetric edge into one canonical direction, so that "A is the
// spouse of B" and "B is the spouse of A" cannot both be stored.
func NewRelationship(organisationID, academicYearID, fromPersonID, toPersonID ids.UUID, relationshipType RelationshipType) (*Relationship, error) {
	return NewRelationshipWithID(ids.New(), organisationID, academicYearID, fromPersonID, toPersonID, relationshipType)
}

// NewRelationshipWithID creates a relationship with a caller-supplied identifier.
func NewRelationshipWithID(id, organisationID, academicYearID, fromPersonID, toPersonID ids.UUID, relationshipType RelationshipType) (*Relationship, error) {
	if organisationID.IsNil() {
		return nil, apperr.ValidationFailed("a relationship must belong to an organisation").
			WithDetail("field", "relationship.organisation_id")
	}
	if academicYearID.IsNil() {
		return nil, apperr.ValidationFailed("a relationship must belong to an academic year").
			WithDetail("field", "relationship.academic_year_id")
	}
	if fromPersonID.IsNil() || toPersonID.IsNil() {
		return nil, apperr.ValidationFailed("a relationship must name two people").
			WithDetail("field", "relationship.persons")
	}
	if fromPersonID == toPersonID {
		return nil, apperr.ValidationFailed("a person cannot be related to themselves").
			WithDetail("field", "relationship.persons").
			WithDetail("id", fromPersonID.String())
	}
	if !relationshipType.Valid() {
		return nil, apperr.ValidationFailed("unknown relationship type").
			WithDetail("field", "relationship.relationship_type").
			WithDetail("value", string(relationshipType))
	}

	// Store a symmetric edge once, in a canonical direction.
	if relationshipType.Symmetric() && fromPersonID.String() > toPersonID.String() {
		fromPersonID, toPersonID = toPersonID, fromPersonID
	}

	return &Relationship{
		ID:             id,
		OrganisationID: organisationID,
		AcademicYearID: academicYearID,
		FromPersonID:   fromPersonID,
		ToPersonID:     toPersonID,
		Type:           relationshipType,
		IsActive:       true,
		Version:        1,
	}, nil
}

// SetRole sets the specific label on the edge, such as MOTHER or FATHER.
func (r *Relationship) SetRole(role RelationshipRole) error {
	trimmed := RelationshipRole(strings.ToUpper(strings.TrimSpace(string(role))))
	if utf8.RuneCountInString(string(trimmed)) > maxRelationshipRoleLength {
		return apperr.ValidationFailed("relationship role is too long").
			WithDetail("field", "relationship.relationship_role").
			WithDetail("maxLength", maxRelationshipRoleLength)
	}
	r.Role = trimmed
	return nil
}

// MakePrimary marks this edge as the one Finance should treat as the billing
// guardian. The database allows at most one primary guardian per student per
// year, so promoting a new one demotes the previous holder first.
func (r *Relationship) MakePrimary() error {
	if r.Type != RelationshipGuardianOf {
		return apperr.ValidationFailed("only a guardian relationship can be primary").
			WithDetail("field", "relationship.is_primary").
			WithDetail("relationshipType", string(r.Type))
	}
	r.IsPrimary = true
	return nil
}

// Deactivate ends the relationship without deleting it, so a statement issued
// last year still resolves the guardian who was on it then.
func (r *Relationship) Deactivate() {
	r.IsActive = false
}

// Activate restores a deactivated relationship.
func (r *Relationship) Activate() {
	r.IsActive = true
}

// MemberRole is how a person sits in a household.
type MemberRole string

// Household membership roles.
const (
	MemberHead      MemberRole = "HEAD"
	MemberPartner   MemberRole = "PARTNER"
	MemberDependent MemberRole = "DEPENDENT"
	MemberOther     MemberRole = "OTHER"
)

// Household is a billing party for one academic year.
//
// It exists because the party that receives an invoice is not always a person: a
// couple pays for three children, and a separated family is two households. It
// deliberately holds no family structure — membership is derived from
// GUARDIAN_OF edges at read time, so a family tree is never stored twice.
type Household struct {
	ID             ids.UUID
	OrganisationID ids.UUID
	AcademicYearID ids.UUID
	Name           string
	BillingEmail   string
	BillingPhone   string
	BillingAddress Address
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Members        []HouseholdMember
}

// HouseholdMember is one person's place in a household.
type HouseholdMember struct {
	HouseholdID   ids.UUID
	PersonID      ids.UUID
	Role          MemberRole
	IsBillingPart bool
	CreatedAt     time.Time
}

// NewHousehold creates a household with a generated identifier.
func NewHousehold(organisationID, academicYearID ids.UUID, name string) (*Household, error) {
	return NewHouseholdWithID(ids.New(), organisationID, academicYearID, name)
}

// NewHouseholdWithID creates a household with a caller-supplied identifier.
func NewHouseholdWithID(id, organisationID, academicYearID ids.UUID, name string) (*Household, error) {
	if organisationID.IsNil() || academicYearID.IsNil() {
		return nil, apperr.ValidationFailed("a household must belong to an organisation and an academic year").
			WithDetail("field", "household.scope")
	}

	household := &Household{
		ID:             id,
		OrganisationID: organisationID,
		AcademicYearID: academicYearID,
		Version:        1,
	}
	if err := household.Rename(name); err != nil {
		return nil, err
	}
	return household, nil
}

// Rename sets the household's billing name, which is what appears on a
// statement. A family is often named after one parent, but the school must be
// able to print whatever the family asks for.
func (h *Household) Rename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperr.ValidationFailed("household name must not be empty").
			WithDetail("field", "household.name")
	}
	if err := checkLength(trimmed, 200, "household.name"); err != nil {
		return err
	}
	h.Name = trimmed
	return nil
}

// SetBillingContact sets where statements for this household are sent. It is
// separate from the members' own addresses because a couple may have a shared
// home address, a PO box, or one parent's work address for school billing.
func (h *Household) SetBillingContact(email, phone string, address Address) error {
	normalised, err := normaliseEmail(email)
	if err != nil {
		return apperr.ValidationFailed("household billing email is not valid").
			WithDetail("field", "household.billing_email")
	}
	if err := checkPhone(phone, "household.billing_phone"); err != nil {
		return err
	}
	if err := checkLength(address.Line1, maxAddressLineLength, "household.billing_address_line1"); err != nil {
		return err
	}

	address.Line1 = strings.TrimSpace(address.Line1)
	address.Line2 = strings.TrimSpace(address.Line2)
	address.City = strings.TrimSpace(address.City)
	address.PostalCode = strings.TrimSpace(address.PostalCode)
	address.Country = strings.ToUpper(strings.TrimSpace(address.Country))

	h.BillingEmail = normalised
	h.BillingPhone = strings.TrimSpace(phone)
	h.BillingAddress = address
	return nil
}

// AddMember places a person in the household.
//
// It returns the stored form of the member, not the one it was given: the
// household stamps its own identifier and normalises an empty role. The caller
// must persist what is returned, or it will write a row that fails the foreign
// key on households, or a role its own CHECK constraint rejects.
func (h *Household) AddMember(member HouseholdMember) (HouseholdMember, error) {
	if member.PersonID.IsNil() {
		return HouseholdMember{}, apperr.ValidationFailed("a household member must be a person").
			WithDetail("field", "household_member.person_id")
	}
	switch member.Role {
	case MemberHead, MemberPartner, MemberDependent, MemberOther:
	case "":
		member.Role = MemberOther
	default:
		return HouseholdMember{}, apperr.ValidationFailed("unknown household member role").
			WithDetail("field", "household_member.member_role").
			WithDetail("value", string(member.Role))
	}

	member.HouseholdID = h.ID
	for _, existing := range h.Members {
		if existing.PersonID == member.PersonID {
			return HouseholdMember{}, apperr.AlreadyExists("household member", "person", member.PersonID.String())
		}
	}
	h.Members = append(h.Members, member)
	return member, nil
}
