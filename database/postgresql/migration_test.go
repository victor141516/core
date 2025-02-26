package postgresql

import (
	"fmt"
	"path"
	"testing"

	"github.com/spf13/afero"
)

func TestMigrateUp(t *testing.T) {
	last, err := getLastMigration()
	if err != nil {
		t.Fatal(err)
	}

	next := last + 1

	fakeMigration := `
		CREATE TABLE IF NOT EXISTS sb.unittests (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			value TEXT NOT NULL
	);
	
	INSERT INTO sb.unittests(value)
	VALUES('yep');
	`

	fakeMigrationFile := fmt.Sprintf("%04d_fake_migration.sql", next)
	if err := afero.WriteFile(appFS, path.Join("./sql", fakeMigrationFile), []byte(fakeMigration), 0664); err != nil {
		t.Fatal(err)
	}

	if err := migrate(datastore.DB); err != nil {
		t.Fatal(err)
	}

	check, err := getDBLastMigration(datastore.DB)
	if err != nil {
		t.Fatal(err)
	} else if next != check {
		t.Errorf("expected last db migration to be %d got %d", next, check)
	}

	var inserted string
	if err := datastore.DB.QueryRow(`SELECT value FROM sb.unittests LIMIT 1`).Scan(&inserted); err != nil {
		t.Fatal(err)
	} else if inserted != "yep" {
		t.Errorf("expected 'yep' from inserted migration value, got %s", inserted)
	}
}
