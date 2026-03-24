package runtime

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestLegacyBackupAliasIsNotServed(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "backups-route.db"), `
app TodoApi

auth {
  email_transport console
  email_from "no-reply@example.com"
  email_subject "Your login code"
}

entity Todo {
  title: String
  authorize all when user_role == "admin"
}
`)

	rec := doRuntimeRequest(r, http.MethodPost, "/_mar/backup", "", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for legacy backup alias, got %d body=%s", rec.Code, rec.Body.String())
	}
}
