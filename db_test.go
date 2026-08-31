package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *testingDB {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_ottie.db")
	db, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("openDB failed: %v", err)
	}
	return &testingDB{DB: db, path: dbPath}
}

type testingDB struct {
	DB   *sql.DB
	path string
}

func TestDBUserAndCascade(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	count, err := countUsers(db)
	if err != nil || count != 0 {
		t.Fatalf("initial countUsers expected 0, got %d, err: %v", count, err)
	}

	// 1. Create admin user
	adminID, err := createUserWithDEK(db, "admin", "hash1", "admin", "encdek1", "salt1")
	if err != nil {
		t.Fatalf("create admin user failed: %v", err)
	}
	if adminID != 1 {
		t.Fatalf("expected adminID 1, got %d", adminID)
	}

	// 2. Create regular user "alice"
	aliceID, err := createUserWithDEK(db, "alice", "hash2", "user", "encdek2", "salt2")
	if err != nil {
		t.Fatalf("create alice failed: %v", err)
	}

	// 3. Insert accounts for alice
	err = insertAccount(db, aliceID, "GitHub", "alice@example.com", "Work", "encsecret1", 6, 30, "SHA1")
	if err != nil {
		t.Fatalf("insertAccount failed: %v", err)
	}
	err = insertAccount(db, aliceID, "AWS", "alice-corp", "Personal", "encsecret2", 6, 30, "SHA1")
	if err != nil {
		t.Fatalf("insertAccount 2 failed: %v", err)
	}

	// 4. Verify alice accounts
	aliceAccounts, err := listAccounts(db, aliceID)
	if err != nil || len(aliceAccounts) != 2 {
		t.Fatalf("expected 2 accounts for alice, got %d", len(aliceAccounts))
	}

	// 5. Admin lists users
	users, err := listUsersForAdmin(db)
	if err != nil || len(users) != 2 {
		t.Fatalf("expected 2 users in admin list, got %d", len(users))
	}
	for _, u := range users {
		if u.Username == "alice" && u.AccountCount != 2 {
			t.Fatalf("expected alice account count 2, got %d", u.AccountCount)
		}
		if u.Username == "admin" && u.AccountCount != 0 {
			t.Fatalf("expected admin account count 0, got %d", u.AccountCount)
		}
	}

	// 6. Delete alice (verify cascade)
	err = deleteUser(db, aliceID)
	if err != nil {
		t.Fatalf("deleteUser alice failed: %v", err)
	}

	remainingAccounts, err := listAccounts(db, aliceID)
	if err != nil || len(remainingAccounts) != 0 {
		t.Fatalf("expected 0 accounts for deleted alice, got %d", len(remainingAccounts))
	}

	totalUsers, totalAccs, err := getSystemStats(db)
	if err != nil || totalUsers != 1 || totalAccs != 0 {
		t.Fatalf("expected 1 user and 0 accounts after delete, got %d users, %d accs", totalUsers, totalAccs)
	}
}
