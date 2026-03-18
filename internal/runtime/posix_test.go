package runtime

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestEntityCRUDSupportsPosixFields(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "posix-crud.db"), `
app TodoApi

entity Event {
  title: String
  starts_at: Posix
  authorize all when true
}
`)

	rec := doRuntimeRequest(r, http.MethodPost, "/events", `{"title":"Launch","starts_at":1742203200000}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response failed: %v body=%s", err, rec.Body.String())
	}
	if created["starts_at"] != float64(1742203200000) {
		t.Fatalf("expected starts_at to round-trip as Unix milliseconds, got %#v", created["starts_at"])
	}

	listRec := doRuntimeRequest(r, http.MethodGet, "/events", "", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode list response failed: %v body=%s", err, listRec.Body.String())
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["starts_at"] != float64(1742203200000) {
		t.Fatalf("expected listed starts_at to round-trip as Unix milliseconds, got %#v", rows[0]["starts_at"])
	}
}

func TestActionsSupportPosixInputFields(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "posix-action.db"), `
app TodoApi

entity Event {
  title: String
  starts_at: Posix
  authorize all when true
}

type alias ScheduleEventInput =
  { starts_at: Posix
  }

action scheduleEvent {
  input: ScheduleEventInput

  create Event {
    title: "Launch"
    starts_at: input.starts_at
  }
}
`)

	rec := doRuntimeRequest(r, http.MethodPost, "/actions/scheduleEvent", `{"starts_at":1742203200000}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	listRec := doRuntimeRequest(r, http.MethodGet, "/events", "", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode list response failed: %v body=%s", err, listRec.Body.String())
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["starts_at"] != float64(1742203200000) {
		t.Fatalf("expected action-created starts_at to round-trip as Unix milliseconds, got %#v", rows[0]["starts_at"])
	}
}

func TestActionsSupportUpdateAndDeleteSteps(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "action-update-delete.db"), `
app TodoApi

entity Todo {
  title: String
  done: Bool default false
  authorize all when true
}

type alias RenameTodoInput =
  { id: Int
  , title: String
  }

type alias DeleteTodoInput =
  { id: Int
  }

action renameTodo {
  input: RenameTodoInput

  update Todo {
    id: input.id
    title: input.title
  }
}

action deleteTodo {
  input: DeleteTodoInput

  delete Todo {
    id: input.id
  }
}
`)

	createRec := doRuntimeRequest(r, http.MethodPost, "/todos", `{"title":"Before"}`, "")
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	updateRec := doRuntimeRequest(r, http.MethodPost, "/actions/renameTodo", `{"id":1,"title":"After"}`, "")
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 update action, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	getRec := doRuntimeRequest(r, http.MethodGet, "/todos/1", "", "")
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 get, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var updated map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode get response failed: %v body=%s", err, getRec.Body.String())
	}
	if updated["title"] != "After" {
		t.Fatalf("expected updated title After, got %#v", updated["title"])
	}

	deleteRec := doRuntimeRequest(r, http.MethodPost, "/actions/deleteTodo", `{"id":1}`, "")
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 delete action, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	missingRec := doRuntimeRequest(r, http.MethodGet, "/todos/1", "", "")
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d body=%s", missingRec.Code, missingRec.Body.String())
	}
}
