package projecttest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	httppkg "github.com/nicoc/socialflow/internal/http"
)

// TestContentHandlerHasNoListByMonth provides versionable automated evidence
// that the dead duplicate ContentHandler.ListByMonth method has been removed.
// Uses reflection — the test FAILS if someone reintroduces the method.
func TestContentHandlerHasNoListByMonth(t *testing.T) {
	// Use pointer type — ContentHandler methods use pointer receivers (*ContentHandler).
	chType := reflect.TypeOf(&httppkg.ContentHandler{})

	t.Run("no ListByMonth method on ContentHandler", func(t *testing.T) {
		for i := range chType.NumMethod() {
			methodName := chType.Method(i).Name
			if methodName == "ListByMonth" {
				t.Errorf(
					"ContentHandler.ListByMonth still exists (%d methods found) — "+
						"this is the dead duplicate method that the refactor removed. "+
						"Use CalendarHandler.ListByMonth (GET /api/calendar) instead.",
					chType.NumMethod(),
				)
				return
			}
		}
		// Positive assertion: the method count is what we expect (no removal gaps).
		if chType.NumMethod() == 0 {
			t.Error("ContentHandler has zero exported methods — struct or svc field may be missing")
		}
	})

	t.Run("ContentHandler struct and svc field remain", func(t *testing.T) {
		elemType := chType.Elem() // deref *ContentHandler → ContentHandler
		if elemType.Kind() != reflect.Struct {
			t.Fatalf("ContentHandler is not a struct: %v", elemType.Kind())
		}
		svcField, ok := elemType.FieldByName("svc")
		if !ok {
			t.Error("ContentHandler.svc field is missing — deleting ListByMonth should not touch the struct")
		}
		if !svcField.IsExported() {
			// unexported is fine — it's internal wiring
			t.Logf("ContentHandler.svc is unexported (expected for internal service wiring)")
		}
	})

	t.Run("active ContentHandler methods still exist", func(t *testing.T) {
		requiredMethods := []string{"List", "Create", "Get", "Update", "TransitionStatus"}
		methodSet := make(map[string]bool)
		for i := range chType.NumMethod() {
			methodSet[chType.Method(i).Name] = true
		}
		for _, m := range requiredMethods {
			if !methodSet[m] {
				t.Errorf("ContentHandler.%s is missing — it should survive the refactor", m)
			}
		}
	})
}

// TestCalendarRouteWiredToCalendarHandler verifies the calendar HTTP route
// remains wired to CalendarHandler.ListByMonth in cmd/api/main.go.
// It reads the file directly — no build required.
func TestCalendarRouteWiredToCalendarHandler(t *testing.T) {
	root := projectRoot(t)
	mainGoPath := filepath.Join(root, "cmd", "api", "main.go")

	data, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("cannot read main.go: %v", err)
	}
	content := string(data)

	t.Run("calendar route uses CalendarHandler.ListByMonth", func(t *testing.T) {
		// The exact line per the spec: r.Get("/calendar", calendarH.ListByMonth)
		wantFragment := `calendarH.ListByMonth`
		if !strings.Contains(content, wantFragment) {
			t.Errorf(
				"cmd/api/main.go does not contain %q — "+
					"the /calendar route may not be wired to CalendarHandler.ListByMonth",
				wantFragment,
			)
		}
	})

	t.Run("no ContentHandler.ListByMonth route binding exists", func(t *testing.T) {
		// If someone rewires the route to ContentHandler, this catches it.
		bindingFragments := []string{
			`contentH.ListByMonth`,
			`contentHandler.ListByMonth`,
			`ContentHandler.ListByMonth`,
		}
		for _, frag := range bindingFragments {
			if strings.Contains(content, frag) {
				t.Errorf(
					"cmd/api/main.go contains %q — route may be wired to the removed ContentHandler method",
					frag,
				)
			}
		}
	})

	t.Run("CalendarHandler struct exists in handler_calendar.go", func(t *testing.T) {
		calPath := filepath.Join(root, "internal", "http", "handler_calendar.go")
		calData, err := os.ReadFile(calPath)
		if err != nil {
			t.Fatalf("cannot read handler_calendar.go: %v", err)
		}
		calContent := string(calData)

		if !strings.Contains(calContent, "CalendarHandler struct") {
			t.Error("CalendarHandler struct not found in handler_calendar.go — may have been accidentally removed")
		}
		if !strings.Contains(calContent, "func (h *CalendarHandler) ListByMonth") {
			t.Error("CalendarHandler.ListByMonth method not found in handler_calendar.go — may have been accidentally removed")
		}
	})
}
