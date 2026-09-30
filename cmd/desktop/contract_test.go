package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
	"github.com/elcomware/edupilot/internal/kernel/ids"
)

// The desktop transport and the frontend gateway are two halves of one contract
// written in two languages, and neither language can see the other. A gateway
// operation naming a method that does not exist, or calling it with the wrong
// argument shape, compiles cleanly in both and fails only at runtime, in a
// shipped desktop build, on the screen a school uses first.
//
// These tests read the frontend gateway and check it against the Go service, so
// the halves cannot drift apart without a test going red.

// operationPattern finds the operation names the gateway calls, which are string
// literals of the form <namespace>.App.<Method>.
var operationPattern = regexp.MustCompile(`'([a-z]+\.[a-zA-Z]+\.App\.[A-Z][A-Za-z]*)'`)

func gatewaySource(t *testing.T) string {
	t.Helper()

	// Resolved from this file's own location rather than the working directory,
	// so the test means the same thing from the module root or from cmd/desktop.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test source")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "frontend", "src", "gateway", "services.ts")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the frontend gateway: %v", err)
	}
	return string(source)
}

func gatewayOperations(t *testing.T) map[string]struct{} {
	t.Helper()

	operations := map[string]struct{}{}
	for _, match := range operationPattern.FindAllStringSubmatch(gatewaySource(t), -1) {
		operations[match[1]] = struct{}{}
	}
	if len(operations) == 0 {
		t.Fatal("no gateway operations were found, so this test would pass on an empty file")
	}
	return operations
}

func methodName(operation string) string {
	parts := strings.Split(operation, ".")
	return parts[len(parts)-1]
}

// callBodies are the argument lists of every call to one operation. An operation
// may be called from more than one place, such as a generic roster read and a
// screen that pins the role it wants.
func callBodies(t *testing.T, source, operation string) []string {
	t.Helper()

	needle := "'" + operation + "'"
	bodies := make([]string, 0, 1)
	for offset := 0; ; {
		start := strings.Index(source[offset:], needle)
		if start < 0 {
			break
		}
		start += offset
		rest := source[start:]
		end := strings.Index(rest, ")")
		if end < 0 {
			t.Fatalf("operation %s has no closing parenthesis", operation)
		}
		bodies = append(bodies, rest[:end])
		offset = start + len(needle)
	}
	if len(bodies) == 0 {
		t.Fatalf("operation %s is not in the gateway", operation)
	}
	return bodies
}

// callBody is the argument list of the first call to one operation.
func callBody(t *testing.T, source, operation string) string {
	t.Helper()
	return callBodies(t, source, operation)[0]
}

// notYetBuilt are namespaces the gateway declares ahead of the Go services behind
// them. They are listed rather than skipped silently: when one is built, removing
// it from this list makes its operations subject to the same checks as the rest,
// and until then nobody is told their finance screen works when it cannot run.
var notYetBuilt = map[string]string{
	"finance.accounting":  "accounting",
	"finance.assets":      "assets",
	"finance.banking":     "banking",
	"finance.billing":     "billing",
	"finance.budgeting":   "budgeting",
	"finance.collections": "collections",
	"finance.inventory":   "inventory",
	"finance.payroll":     "payroll",
	"finance.procurement": "procurement",
	"finance.receivables": "receivables",
	"finance.reporting":   "reporting",
}

// The heart of it: every operation the gateway can call must name a method that
// really exists. This is the test that caught platform.people.App.ListStudents, a
// Students screen calling a transport method nobody ever wrote.
func TestEveryGatewayOperationIsABoundMethod(t *testing.T) {
	service := reflect.TypeOf(&App{})

	for operation := range gatewayOperations(t) {
		namespace := strings.TrimSuffix(operation, ".App."+methodName(operation))
		if _, planned := notYetBuilt[namespace]; planned {
			continue
		}
		if _, found := service.MethodByName(methodName(operation)); !found {
			t.Errorf("the gateway calls %s but App has no method %s", operation, methodName(operation))
		}
	}
}

// Every operation must belong to a namespace that is either built or declared as
// not yet built, or the screen is a placeholder presented as a feature.
func TestNoOperationIsOutsideTheBuiltAndNotYetBuiltNamespaces(t *testing.T) {
	built := map[string]struct{}{
		"platform.organisation": {},
		"platform.campus":       {},
		"platform.academic":     {},
		"platform.people":       {},
	}

	for operation := range gatewayOperations(t) {
		namespace := strings.TrimSuffix(operation, ".App."+methodName(operation))
		if _, ok := built[namespace]; ok {
			continue
		}
		if _, planned := notYetBuilt[namespace]; !planned {
			t.Errorf("%s uses namespace %s, which is neither built nor declared as not yet built", operation, namespace)
		}
	}
}

// Wails binds arguments positionally: the frontend sends one argument per Go
// parameter other than the context, so a method with two request parameters can
// never be reached at all. And a bare string parameter is the same bug wearing a
// different hat, because it forces the gateway to call that one method with a
// string and its neighbours with objects. One object in, one envelope out.
func TestEveryBoundMethodTakesAtMostOneRequest(t *testing.T) {
	service := reflect.TypeOf(&App{})
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	for i := 0; i < service.NumMethod(); i++ {
		method := service.Method(i)
		t.Run(method.Name, func(t *testing.T) {
			// In(0) is the receiver and the context is supplied by the runtime, so
			// neither is bound. What the frontend has to fill is what is left.
			bound := make([]reflect.Type, 0, 2)
			for parameter := 1; parameter < method.Type.NumIn(); parameter++ {
				if method.Type.In(parameter) == contextType {
					continue
				}
				bound = append(bound, method.Type.In(parameter))
			}

			switch len(bound) {
			case 0:
				return
			case 1:
			default:
				t.Fatalf("takes %d bound arguments, want at most 1: Wails sends one argument per parameter, so further parameters can never be filled", len(bound))
			}
			if bound[0].Kind() != reflect.Struct {
				t.Errorf("takes %s, want a request struct: a bare value forces the gateway to call this method differently from every other one", bound[0])
			}
		})
	}
}

// A method nobody calls is still part of the transport's surface: bound, reachable
// from the frontend, untested. If nothing calls it, it should not be exported.
func TestEveryBoundMethodIsCalledByTheGateway(t *testing.T) {
	source := gatewaySource(t)
	service := reflect.TypeOf(&App{})

	for i := 0; i < service.NumMethod(); i++ {
		name := service.Method(i).Name
		if !strings.Contains(source, ".App."+name+"'") {
			t.Errorf("App.%s is bound but the gateway never calls it; unexport it or call it", name)
		}
	}
}

// A request must be passed on its own, never wrapped in a field. Wrapping it hands
// the transport an object whose fields it cannot see, so the page silently comes
// back empty instead of failing loudly.
func TestGatewayDoesNotWrapRequestObjects(t *testing.T) {
	source := gatewaySource(t)

	for _, match := range operationPattern.FindAllStringSubmatch(source, -1) {
		if body := callBody(t, source, match[1]); strings.Contains(body, "request:") {
			t.Errorf("%s sends { request: ... } but the transport takes the request object itself", match[1])
		}
	}
}

// Where the gateway builds a request inline, the fields it sends must be the ones
// the transport parses, so a renamed field is caught here rather than as a person
// who cannot be opened and no error to explain why. Operations that pass a typed
// request straight through are not listed: their field names live in the request
// type, and the type checker, not this test, is what keeps those two in step.
func TestInlineGatewayRequestsCarryTheFieldsTheTransportParses(t *testing.T) {
	source := gatewaySource(t)

	for operation, fields := range map[string][]string{
		"platform.people.App.GetPerson":          {"id"},
		"platform.people.App.Unlink":             {"id"},
		"platform.academic.App.ListTerms":        {"academicYearId"},
		"platform.people.App.ListHouseholds":     {"academicYearId"},
		"platform.people.App.ListPeopleWithRole": {"role"},
	} {
		bodies := callBodies(t, source, operation)
		missing := make([]string, 0, len(fields))
		for _, field := range fields {
			sent := false
			for _, body := range bodies {
				if strings.Contains(body, field) {
					sent = true
					break
				}
			}
			if !sent {
				missing = append(missing, field)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			t.Errorf("%s never sends %s", operation, strings.Join(missing, ", "))
		}
	}
}

// A malformed identifier is a validation failure and a well-formed one that
// matches nothing is NOT_FOUND. The interface renders these differently, so
// collapsing them sends the user hunting for a record that never existed.
func TestMalformedIdentifiersAreNotReportedAsMissingRecords(t *testing.T) {
	malformed := apperr.CodeOf(api.InvalidID("id", "not-a-uuid"))
	if malformed == apperr.CodeNotFound {
		t.Errorf("a malformed identifier reports %q, want %q: the interface tells these two apart, and NOT_FOUND sends the user looking for a record that never existed",
			malformed, apperr.CodeValidationFailed)
	}

	absent := apperr.CodeOf(api.NotFoundEntity("person", ids.New().String()))
	if absent != apperr.CodeNotFound {
		t.Errorf("a well-formed identifier that matches nothing reports %q, want %q", absent, apperr.CodeNotFound)
	}
}
