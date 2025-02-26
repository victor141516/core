package mongo

import (
	"fmt"
	"testing"
)

func TestFindAccount(t *testing.T) {
	cus, err := datastore.FindAccount(dbTest.CustomerID)
	if err != nil {
		t.Fatal(err)
	} else if cus.ID != dbTest.CustomerID {
		t.Errorf("expected customer id to be %s got %s", dbTest.CustomerID, cus.ID)
	}
}

func TestFindDatabase(t *testing.T) {
	b, err := datastore.FindDatabase(dbTest.ID)
	if err != nil {
		t.Fatal(err)
	} else if b.Name != dbTest.Name {
		fmt.Errorf("expected name to be %s got %s", dbTest.Name, b.Name)
	}
}

func TestDatabaseExists(t *testing.T) {
	exists, err := datastore.DatabaseExists(dbTest.Name)
	if err != nil {
		t.Fatal(err)
	} else if !exists {
		t.Fatal("database should exists")
	}
}

func TestListDatabases(t *testing.T) {
	dbs, err := datastore.ListDatabases()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, db := range dbs {
		if db.ID == dbTest.ID {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("test db should be part of the db list")
	}
}

func TestIncrementMonthlyEmailSent(t *testing.T) {
	if err := datastore.IncrementMonthlyEmailSent(dbTest.ID); err != nil {
		t.Fatal(err)
	}

	expected := dbTest.MonthlySentEmail + 1

	b, err := datastore.FindDatabase(dbTest.ID)
	if err != nil {
		t.Fatal(err)
	} else if b.MonthlySentEmail != expected {
		t.Errorf("expected monthly sent to be %d got %d", expected, b.MonthlySentEmail)
	}
}

func TestGetCustomerByStripeID(t *testing.T) {
	cus, err := datastore.GetCustomerByStripeID(adminEmail)
	if err != nil {
		t.Fatal(err)
	} else if cus.ID != dbTest.CustomerID {
		t.Errorf("exepected cus to have id %s got %s", dbTest.CustomerID, cus.ID)
	}
}

func TestActivateCustomer(t *testing.T) {
	if err := datastore.ActivateCustomer(dbTest.CustomerID); err != nil {
		t.Fatal(err)
	}

	cus, err := datastore.FindAccount(dbTest.CustomerID)
	if err != nil {
		t.Fatal(err)
	} else if !cus.IsActive {
		t.Errorf("expected cus to be active")
	}
}

func TestNewID(t *testing.T) {
	id1 := datastore.NewID()
	id2 := datastore.NewID()

	if len(id1) < 10 {
		t.Errorf("expected new id to be > than 10, got %s", id1)
	} else if id1 == id2 {
		t.Errorf("expected id to be different got 1: %s 2: %s", id1, id2)
	}
}
