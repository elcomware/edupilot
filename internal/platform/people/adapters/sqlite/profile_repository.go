package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/elcomware/edupilot/internal/infrastructure/database"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
	"github.com/elcomware/edupilot/internal/platform/people/domain"
)

// StudentProfileRepository stores student facts in SQLite.
type StudentProfileRepository struct {
	transactor *database.Transactor
}

// NewStudentProfileRepository builds the SQLite student profile repository.
func NewStudentProfileRepository(transactor *database.Transactor) *StudentProfileRepository {
	return &StudentProfileRepository{transactor: transactor}
}

// Upsert writes a student profile, keyed by the person role it belongs to.
func (r *StudentProfileRepository) Upsert(ctx context.Context, profile *domain.StudentProfile) error {
	const statement = `INSERT INTO student_profiles
			(person_role_id, student_number, admission_date, status, previous_school, is_boarding, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (person_role_id) DO UPDATE SET
			student_number = excluded.student_number,
			admission_date = excluded.admission_date,
			status = excluded.status,
			previous_school = excluded.previous_school,
			is_boarding = excluded.is_boarding,
			updated_at = excluded.updated_at`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		profile.PersonRoleID.String(),
		profile.StudentNumber,
		optionalDate(profile.AdmissionDate),
		string(profile.Status),
		profile.PreviousSchool,
		profile.IsBoarding,
		profile.CreatedAt.Format(time.RFC3339Nano),
		profile.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if database.IsUniqueViolation(err) && profile.StudentNumber != "" {
			return apperr.AlreadyExists("student", "student number", profile.StudentNumber)
		}
		return apperr.Internal(err, "student profile could not be saved")
	}
	return nil
}

// ByRoleID loads the student profile of a role, or nil.
func (r *StudentProfileRepository) ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.StudentProfile, error) {
	const query = `SELECT person_role_id, student_number, admission_date, status,
		previous_school, is_boarding, created_at, updated_at
		FROM student_profiles WHERE person_role_id = ?`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, roleID.String())
	if err != nil {
		return nil, apperr.Internal(err, "student profile lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "student profile lookup failed")
		}
		return nil, nil
	}
	return scanStudentProfile(rows)
}

// FindByNumber returns the student profile with a student number, or nil. The
// lookup is not organisation-scoped because a student number is unique across
// the installation by the partial unique index.
func (r *StudentProfileRepository) FindByNumber(ctx context.Context, number string) (*domain.StudentProfile, error) {
	if number == "" {
		return nil, nil
	}
	const query = `SELECT person_role_id, student_number, admission_date, status,
		previous_school, is_boarding, created_at, updated_at
		FROM student_profiles WHERE student_number = ?`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, number)
	if err != nil {
		return nil, apperr.Internal(err, "student number lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "student number lookup failed")
		}
		return nil, nil
	}
	return scanStudentProfile(rows)
}

func scanStudentProfile(row interface{ Scan(...any) error }) (*domain.StudentProfile, error) {
	var (
		rawRoleID, number, status, previousSchool string
		admissionDate                             sql.NullString
		isBoarding                                bool
		rawCreatedAt, rawUpdatedAt                string
	)

	err := row.Scan(&rawRoleID, &number, &admissionDate, &status, &previousSchool,
		&isBoarding, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "student profile could not be read")
	}

	roleID, err := ids.Parse(rawRoleID)
	if err != nil {
		return nil, apperr.Internal(err, "student profile role identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "student profile creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "student profile update timestamp is invalid")
	}

	profile := &domain.StudentProfile{
		PersonRoleID:   roleID,
		StudentNumber:  number,
		Status:         domain.StudentStatus(status),
		PreviousSchool: previousSchool,
		IsBoarding:     isBoarding,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}
	if admissionDate.Valid && admissionDate.String != "" {
		parsed, err := time.Parse(dateLayout, admissionDate.String)
		if err != nil {
			return nil, apperr.Internal(err, "student admission date is invalid")
		}
		profile.AdmissionDate = &parsed
	}
	return profile, nil
}

// EmployeeProfileRepository stores employee facts in SQLite.
type EmployeeProfileRepository struct {
	transactor *database.Transactor
}

// NewEmployeeProfileRepository builds the SQLite employee profile repository.
func NewEmployeeProfileRepository(transactor *database.Transactor) *EmployeeProfileRepository {
	return &EmployeeProfileRepository{transactor: transactor}
}

// Upsert writes an employee profile, keyed by the person role it belongs to.
func (r *EmployeeProfileRepository) Upsert(ctx context.Context, profile *domain.EmployeeProfile) error {
	const statement = `INSERT INTO employee_profiles
			(person_role_id, employee_number, job_title, department, hired_on, ended_on,
			 payroll_group, contract_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (person_role_id) DO UPDATE SET
			employee_number = excluded.employee_number,
			job_title = excluded.job_title,
			department = excluded.department,
			hired_on = excluded.hired_on,
			ended_on = excluded.ended_on,
			payroll_group = excluded.payroll_group,
			contract_type = excluded.contract_type,
			updated_at = excluded.updated_at`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		profile.PersonRoleID.String(),
		profile.EmployeeNumber,
		profile.JobTitle,
		profile.Department,
		optionalDate(profile.HiredOn),
		optionalDate(profile.EndedOn),
		profile.PayrollGroup,
		string(profile.ContractType),
		profile.CreatedAt.Format(time.RFC3339Nano),
		profile.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if database.IsUniqueViolation(err) && profile.EmployeeNumber != "" {
			return apperr.AlreadyExists("employee", "employee number", profile.EmployeeNumber)
		}
		return apperr.Internal(err, "employee profile could not be saved")
	}
	return nil
}

// ByRoleID loads the employee profile of a role, or nil.
func (r *EmployeeProfileRepository) ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.EmployeeProfile, error) {
	const query = `SELECT person_role_id, employee_number, job_title, department, hired_on,
		ended_on, payroll_group, contract_type, created_at, updated_at
		FROM employee_profiles WHERE person_role_id = ?`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, roleID.String())
	if err != nil {
		return nil, apperr.Internal(err, "employee profile lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "employee profile lookup failed")
		}
		return nil, nil
	}
	return scanEmployeeProfile(rows)
}

// FindByNumber returns the employee profile with an employee number, or nil.
func (r *EmployeeProfileRepository) FindByNumber(ctx context.Context, number string) (*domain.EmployeeProfile, error) {
	if number == "" {
		return nil, nil
	}
	const query = `SELECT person_role_id, employee_number, job_title, department, hired_on,
		ended_on, payroll_group, contract_type, created_at, updated_at
		FROM employee_profiles WHERE employee_number = ?`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, number)
	if err != nil {
		return nil, apperr.Internal(err, "employee number lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "employee number lookup failed")
		}
		return nil, nil
	}
	return scanEmployeeProfile(rows)
}

func scanEmployeeProfile(row interface{ Scan(...any) error }) (*domain.EmployeeProfile, error) {
	var (
		rawRoleID, number, jobTitle, department, payrollGroup, contractType string
		hiredOn, endedOn                                                    sql.NullString
		rawCreatedAt, rawUpdatedAt                                          string
	)

	err := row.Scan(&rawRoleID, &number, &jobTitle, &department, &hiredOn, &endedOn,
		&payrollGroup, &contractType, &rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "employee profile could not be read")
	}

	roleID, err := ids.Parse(rawRoleID)
	if err != nil {
		return nil, apperr.Internal(err, "employee profile role identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "employee profile creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "employee profile update timestamp is invalid")
	}

	profile := &domain.EmployeeProfile{
		PersonRoleID:   roleID,
		EmployeeNumber: number,
		JobTitle:       jobTitle,
		Department:     department,
		PayrollGroup:   payrollGroup,
		ContractType:   domain.ContractType(contractType),
		CreatedAt:      created,
		UpdatedAt:      updated,
	}
	if hiredOn.Valid && hiredOn.String != "" {
		parsed, err := time.Parse(dateLayout, hiredOn.String)
		if err != nil {
			return nil, apperr.Internal(err, "employee hire date is invalid")
		}
		profile.HiredOn = &parsed
	}
	if endedOn.Valid && endedOn.String != "" {
		parsed, err := time.Parse(dateLayout, endedOn.String)
		if err != nil {
			return nil, apperr.Internal(err, "employee end date is invalid")
		}
		profile.EndedOn = &parsed
	}
	return profile, nil
}

// GuardianProfileRepository stores guardian rights in SQLite.
type GuardianProfileRepository struct {
	transactor *database.Transactor
}

// NewGuardianProfileRepository builds the SQLite guardian profile repository.
func NewGuardianProfileRepository(transactor *database.Transactor) *GuardianProfileRepository {
	return &GuardianProfileRepository{transactor: transactor}
}

// Upsert writes a guardian profile, keyed by the person role it belongs to.
func (r *GuardianProfileRepository) Upsert(ctx context.Context, profile *domain.GuardianProfile) error {
	const statement = `INSERT INTO guardian_profiles
			(person_role_id, is_emergency_contact, may_collect_student, is_billing_contact,
			 can_authorise_medical, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (person_role_id) DO UPDATE SET
			is_emergency_contact = excluded.is_emergency_contact,
			may_collect_student = excluded.may_collect_student,
			is_billing_contact = excluded.is_billing_contact,
			can_authorise_medical = excluded.can_authorise_medical,
			updated_at = excluded.updated_at`

	_, err := r.transactor.Conn(ctx).ExecContext(ctx, statement,
		profile.PersonRoleID.String(),
		profile.IsEmergencyContact,
		profile.MayCollectStudent,
		profile.IsBillingContact,
		profile.CanAuthoriseMedical,
		profile.CreatedAt.Format(time.RFC3339Nano),
		profile.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return apperr.Internal(err, "guardian profile could not be saved")
	}
	return nil
}

// ByRoleID loads the guardian profile of a role, or nil.
func (r *GuardianProfileRepository) ByRoleID(ctx context.Context, roleID ids.UUID) (*domain.GuardianProfile, error) {
	const query = `SELECT person_role_id, is_emergency_contact, may_collect_student,
		is_billing_contact, can_authorise_medical, created_at, updated_at
		FROM guardian_profiles WHERE person_role_id = ?`

	rows, err := r.transactor.Conn(ctx).QueryContext(ctx, query, roleID.String())
	if err != nil {
		return nil, apperr.Internal(err, "guardian profile lookup failed")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, apperr.Internal(err, "guardian profile lookup failed")
		}
		return nil, nil
	}
	return scanGuardianProfile(rows)
}

func scanGuardianProfile(row interface{ Scan(...any) error }) (*domain.GuardianProfile, error) {
	var (
		rawRoleID                  string
		emergency, mayCollect      bool
		isBilling, mayAuthorise    bool
		rawCreatedAt, rawUpdatedAt string
	)

	err := row.Scan(&rawRoleID, &emergency, &mayCollect, &isBilling, &mayAuthorise,
		&rawCreatedAt, &rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "guardian profile could not be read")
	}

	roleID, err := ids.Parse(rawRoleID)
	if err != nil {
		return nil, apperr.Internal(err, "guardian profile role identifier is invalid")
	}
	created, err := time.Parse(time.RFC3339, rawCreatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "guardian profile creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339, rawUpdatedAt)
	if err != nil {
		return nil, apperr.Internal(err, "guardian profile update timestamp is invalid")
	}

	return &domain.GuardianProfile{
		PersonRoleID:        roleID,
		IsEmergencyContact:  emergency,
		MayCollectStudent:   mayCollect,
		IsBillingContact:    isBilling,
		CanAuthoriseMedical: mayAuthorise,
		CreatedAt:           created,
		UpdatedAt:           updated,
	}, nil
}
