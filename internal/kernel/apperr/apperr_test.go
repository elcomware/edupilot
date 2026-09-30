package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewCarriesCode(t *testing.T) {
	err := New(CodeNotFound, "campus 42 was not found")

	if got := CodeOf(err); got != CodeNotFound {
		t.Fatalf("CodeOf = %s, want %s", got, CodeNotFound)
	}
	if !Is(err, CodeNotFound) {
		t.Fatal("Is(CodeNotFound) = false")
	}
	if Is(err, CodeConflict) {
		t.Fatal("Is(CodeConflict) = true")
	}
}

func TestWrapPreservesCause(t *testing.T) {
	sentinel := errors.New("sqlite is locked")
	err := Wrap(sentinel, CodeInternal, "could not write the audit record")

	if !errors.Is(err, sentinel) {
		t.Fatal("errors.Is could not find the cause")
	}
	if got := CodeOf(err); got != CodeInternal {
		t.Fatalf("CodeOf = %s, want INTERNAL", got)
	}
}

func TestUnknownErrorBecomesInternalWithoutLeaking(t *testing.T) {
	leaky := errors.New("no such table: finance_invoice at /home/school/edupilot.db")

	wire := FromError(leaky)
	if wire.Code != CodeInternal {
		t.Fatalf("code = %s, want INTERNAL", wire.Code)
	}
	if wire.Message == leaky.Error() {
		t.Fatal("the internal error message leaked the cause text")
	}
}

func TestNotFoundIncludesDetails(t *testing.T) {
	err := NotFound("campus", "0193f1c0-0000-7000-8000-000000000001")

	details := err.Details()
	if details["entity"] != "campus" {
		t.Fatalf("details.entity = %v, want campus", details["entity"])
	}
	if details["id"] != "0193f1c0-0000-7000-8000-000000000001" {
		t.Fatalf("details.id = %v", details["id"])
	}
}

func TestMarshalJSONHidesCause(t *testing.T) {
	err := Wrap(errors.New("secret sql"), CodePeriodClosed, "period 2026-01 is closed")

	data, marshalErr := err.MarshalJSON()
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if got, want := string(data), `{"code":"PERIOD_CLOSED","message":"period 2026-01 is closed"}`; got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func TestInsufficientPermissionNamesThePermission(t *testing.T) {
	err := InsufficientPermission("close an accounting period", "accounting.period.close")

	if err.Details()["permission"] != "accounting.period.close" {
		t.Fatalf("details = %v", err.Details())
	}
}

func TestErrorStringIncludesCode(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", New(CodeValidationFailed, "amount must be positive"))
	if got := err.Error(); got == "" {
		t.Fatal("empty error string")
	}
	if !Is(err, CodeValidationFailed) {
		t.Fatal("Is did not see through the wrap")
	}
}
