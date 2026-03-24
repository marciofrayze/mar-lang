package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
)

func TestEntityCRUDSupportsBelongsToFields(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "belongs-to-crud.db"), `
app StoreApi

entity Book {
  title: String
  authorize all when true
}

entity Review {
  body: String
  belongs_to Book
  authorize all when true
}
`)

	bookRec := doRuntimeRequest(r, http.MethodPost, "/books", `{"title":"DDD"}`, "")
	if bookRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating book, got %d body=%s", bookRec.Code, bookRec.Body.String())
	}

	reviewRec := doRuntimeRequest(r, http.MethodPost, "/reviews", `{"body":"Great","book":1}`, "")
	if reviewRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating review, got %d body=%s", reviewRec.Code, reviewRec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(reviewRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response failed: %v body=%s", err, reviewRec.Body.String())
	}
	if created["book"] != float64(1) {
		t.Fatalf("expected review response to expose logical belongs_to field, got %#v", created["book"])
	}

	row, ok, err := queryRow(r.DB, `SELECT book_id FROM reviews WHERE id = 1`)
	if err != nil {
		t.Fatalf("query review row failed: %v", err)
	}
	if !ok {
		t.Fatal("expected review row to exist")
	}
	if row["book_id"] != int64(1) {
		t.Fatalf("expected stored foreign key column book_id=1, got %#v", row["book_id"])
	}
}

func TestEntityCRUDSupportsManyToManyViaJoinEntityBelongsTo(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "belongs-to-join.db"), `
app EnrollmentApi

entity Student {
  name: String
  authorize all when true
}

entity Course {
  title: String
  authorize all when true
}

entity Enrollment {
  belongs_to Student
  belongs_to Course
  authorize all when true
}
`)

	if rec := doRuntimeRequest(r, http.MethodPost, "/students", `{"name":"Mia"}`, ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating student, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRuntimeRequest(r, http.MethodPost, "/courses", `{"title":"Math"}`, ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating course, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec := doRuntimeRequest(r, http.MethodPost, "/enrollments", `{"student":1,"course":1}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating enrollment, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response failed: %v body=%s", err, rec.Body.String())
	}
	if created["student"] != float64(1) || created["course"] != float64(1) {
		t.Fatalf("expected join entity response to expose logical belongs_to fields, got %#v", created)
	}
}

func TestReadAuthorizationFiltersListAndProtectsGet(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "read-filter.db"), `
app TodoReadFilter

auth {
  email_transport console
}

entity Todo {
  title: String
  belongs_to User

  authorize read when user_authenticated and (user == user_id or user_role == "admin")
  authorize create when user_authenticated and user == user_id
  authorize update when user_authenticated and (user == user_id or user_role == "admin")
  authorize delete when user_authenticated and (user == user_id or user_role == "admin")
}
`)

	adminCode := requestCodeAndUseKnownCode(t, r, "owner@example.com")
	adminToken := loginWithCodeAndReadToken(t, r, "owner@example.com", adminCode)

	memberCode := requestCodeAndUseKnownCode(t, r, "member@example.com")
	memberToken := loginWithCodeAndReadToken(t, r, "member@example.com", memberCode)

	adminRow, found, err := r.loadAuthUserByEmail("", "owner@example.com")
	if err != nil {
		t.Fatalf("load admin user failed: %v", err)
	}
	if !found {
		t.Fatal("expected admin user to exist")
	}
	memberRow, found, err := r.loadAuthUserByEmail("", "member@example.com")
	if err != nil {
		t.Fatalf("load member user failed: %v", err)
	}
	if !found {
		t.Fatal("expected member user to exist")
	}

	adminID := adminRow[r.authUser.PrimaryKey]
	memberID := memberRow[r.authUser.PrimaryKey]

	if rec := doRuntimeRequest(r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"Admin todo","user":%v}`, adminID), adminToken); rec.Code != http.StatusCreated {
		t.Fatalf("expected admin todo create to succeed, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRuntimeRequest(r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"Member todo","user":%v}`, memberID), memberToken); rec.Code != http.StatusCreated {
		t.Fatalf("expected member todo create to succeed, got %d body=%s", rec.Code, rec.Body.String())
	}

	memberListRec := doRuntimeRequest(r, http.MethodGet, "/todos", "", memberToken)
	if memberListRec.Code != http.StatusOK {
		t.Fatalf("expected member todo list to succeed, got %d body=%s", memberListRec.Code, memberListRec.Body.String())
	}
	var memberRows []map[string]any
	if err := json.Unmarshal(memberListRec.Body.Bytes(), &memberRows); err != nil {
		t.Fatalf("decode member list failed: %v body=%s", err, memberListRec.Body.String())
	}
	if len(memberRows) != 1 || memberRows[0]["title"] != "Member todo" {
		t.Fatalf("expected member list to be filtered to own row, got %#v", memberRows)
	}

	adminListRec := doRuntimeRequest(r, http.MethodGet, "/todos", "", adminToken)
	if adminListRec.Code != http.StatusOK {
		t.Fatalf("expected admin todo list to succeed, got %d body=%s", adminListRec.Code, adminListRec.Body.String())
	}
	var adminRows []map[string]any
	if err := json.Unmarshal(adminListRec.Body.Bytes(), &adminRows); err != nil {
		t.Fatalf("decode admin list failed: %v body=%s", err, adminListRec.Body.String())
	}
	if len(adminRows) != 2 {
		t.Fatalf("expected admin list to see all rows, got %#v", adminRows)
	}

	memberGetAdminRec := doRuntimeRequest(r, http.MethodGet, fmt.Sprintf("/todos/%v", adminRows[1]["id"]), "", memberToken)
	if memberGetAdminRec.Code != http.StatusForbidden {
		t.Fatalf("expected member get on foreign todo to be forbidden, got %d body=%s", memberGetAdminRec.Code, memberGetAdminRec.Body.String())
	}
}
