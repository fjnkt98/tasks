package ent

import "testing"

func TestSetupClient(t *testing.T) {
	dsn := "file::memory:"
	client, err := SetupClient(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})

	_, err = client.Task.Create().SetTitle("test").SetUserID(999).Save(t.Context())
	if !IsConstraintError(err) {
		t.Fatalf("expected constraint error, but got %v", err)
	}
}

func TestEnforceForeignKeys(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Value string
		Want  string
	}{
		{Name: "memory", Value: "file::memory:", Want: "file::memory:?_fk=1"},
		{Name: "file", Value: "file:app.db", Want: "file:app.db?_fk=1"},
		{Name: "disabled", Value: "file:app.db?_fk=0&_foreign_keys=0", Want: "file:app.db?_fk=1"},
		{Name: "other options", Value: "file:app.db?mode=rwc", Want: "file:app.db?_fk=1&mode=rwc"},
		{Name: "duplicated", Value: "file:app.db?_fk=0&_fk=0", Want: "file:app.db?_fk=1"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			got, err := enforceForeignKeys(test.Value)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.Want {
				t.Errorf("expected '%s', but got '%s'", test.Want, got)
			}
		})
	}

	for _, test := range []struct {
		Name  string
		Value string
	}{
		{Name: "invalid query", Value: "file:app.db?mode=%zz"},
		{Name: "invalid url", Value: "file:///app%zz.d"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, err := enforceForeignKeys(test.Value)
			if err == nil {
				t.Fatal("expected error, but got nil")
			}
		})
	}
}
