package main

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id                 INTEGER PRIMARY KEY AUTOINCREMENT,
	username           TEXT UNIQUE NOT NULL COLLATE NOCASE,
	password_hash      TEXT NOT NULL,
	role               TEXT NOT NULL DEFAULT 'user',
	enc_dek            TEXT NOT NULL,
	salt               TEXT NOT NULL,
	recovery_enc_dek   TEXT DEFAULT '',
	recovery_salt      TEXT DEFAULT '',
	recovery_code_hash TEXT DEFAULT '',
	email              TEXT DEFAULT '',
	phone              TEXT DEFAULT '',
	otp_method         TEXT DEFAULT 'none',
	created_at         DATETIME NOT NULL,
	last_login_at      DATETIME
);

CREATE TABLE IF NOT EXISTS totp_accounts (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	issuer           TEXT NOT NULL,
	account_name     TEXT NOT NULL,
	category         TEXT NOT NULL DEFAULT 'Personal',
	encrypted_secret TEXT NOT NULL,
	digits           INTEGER NOT NULL DEFAULT 6,
	period           INTEGER NOT NULL DEFAULT 30,
	algorithm        TEXT NOT NULL DEFAULT 'SHA1',
	created_at       DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS login_attempts (
	ip           TEXT PRIMARY KEY,
	failures     INTEGER NOT NULL DEFAULT 0,
	locked_until DATETIME
);

CREATE TABLE IF NOT EXISTS user_sessions (
	id           TEXT PRIMARY KEY,
	user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	ip_address   TEXT NOT NULL DEFAULT '',
	user_agent   TEXT NOT NULL DEFAULT '',
	device_name  TEXT NOT NULL DEFAULT '',
	created_at   DATETIME NOT NULL,
	last_seen_at DATETIME NOT NULL,
	revoked_at   DATETIME
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
`

func openDB(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	// Safe migrations for existing databases
	_, _ = db.Exec(`ALTER TABLE totp_accounts ADD COLUMN category TEXT NOT NULL DEFAULT 'Personal';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN recovery_enc_dek TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN recovery_salt TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN recovery_code_hash TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN email TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN phone TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN otp_method TEXT DEFAULT 'none';`)
	return db, nil
}

type User struct {
	ID               int64
	Username         string
	PasswordHash     string
	Role             string
	EncDEK           string
	Salt             string
	RecoveryEncDEK   string
	RecoverySalt     string
	RecoveryCodeHash string
	Email            string
	Phone            string
	OTPMethod        string
	CreatedAt        time.Time
	LastLoginAt      *time.Time
}

type UserAdminView struct {
	ID           int64
	Username     string
	Role         string
	Email        string
	OTPMethod    string
	AccountCount int
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

func countUsers(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func getUserByUsername(db *sql.DB, username string) (*User, error) {
	row := db.QueryRow(`SELECT id, username, password_hash, role, enc_dek, salt, COALESCE(recovery_enc_dek, ''), COALESCE(recovery_salt, ''), COALESCE(recovery_code_hash, ''), COALESCE(email, ''), COALESCE(phone, ''), COALESCE(otp_method, 'none'), created_at, last_login_at FROM users WHERE username = ?`, username)
	u := &User{}
	var lastLogin sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.EncDEK, &u.Salt, &u.RecoveryEncDEK, &u.RecoverySalt, &u.RecoveryCodeHash, &u.Email, &u.Phone, &u.OTPMethod, &u.CreatedAt, &lastLogin); err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return u, nil
}

func getUserByID(db *sql.DB, id int64) (*User, error) {
	row := db.QueryRow(`SELECT id, username, password_hash, role, enc_dek, salt, COALESCE(recovery_enc_dek, ''), COALESCE(recovery_salt, ''), COALESCE(recovery_code_hash, ''), COALESCE(email, ''), COALESCE(phone, ''), COALESCE(otp_method, 'none'), created_at, last_login_at FROM users WHERE id = ?`, id)
	u := &User{}
	var lastLogin sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.EncDEK, &u.Salt, &u.RecoveryEncDEK, &u.RecoverySalt, &u.RecoveryCodeHash, &u.Email, &u.Phone, &u.OTPMethod, &u.CreatedAt, &lastLogin); err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return u, nil
}

func updateUserRecoveryData(db *sql.DB, userID int64, recoveryEncDEK, recoverySalt, phraseHash string) error {
	_, err := db.Exec(`UPDATE users SET recovery_enc_dek = ?, recovery_salt = ?, recovery_code_hash = ? WHERE id = ?`,
		recoveryEncDEK, recoverySalt, phraseHash, userID)
	return err
}

func setUserRecoveryCode(db *sql.DB, userID int64, codeHash string) error {
	_, err := db.Exec(`UPDATE users SET recovery_code_hash = ? WHERE id = ?`, codeHash, userID)
	return err
}

func updateUserContactAndOTP(db *sql.DB, userID int64, email, phone, otpMethod string) error {
	if otpMethod != "email" && otpMethod != "sms" {
		otpMethod = "none"
	}
	_, err := db.Exec(`UPDATE users SET email = ?, phone = ?, otp_method = ? WHERE id = ?`, email, phone, otpMethod, userID)
	return err
}

func createUserWithDEK(db *sql.DB, username, passwordHash, role, encDEK, salt string) (int64, error) {
	res, err := db.Exec(`INSERT INTO users (username, password_hash, role, enc_dek, salt, recovery_code_hash, email, phone, otp_method, created_at) VALUES (?, ?, ?, ?, ?, '', '', '', 'none', ?)`,
		username, passwordHash, role, encDEK, salt, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateLastLogin(db *sql.DB, userID int64) error {
	_, err := db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now().UTC(), userID)
	return err
}

func updateUserPasswordAndDEK(db *sql.DB, userID int64, newPasswordHash, newEncDEK, newSalt string) error {
	_, err := db.Exec(`UPDATE users SET password_hash = ?, enc_dek = ?, salt = ? WHERE id = ?`,
		newPasswordHash, newEncDEK, newSalt, userID)
	return err
}

func listUsersForAdmin(db *sql.DB) ([]UserAdminView, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.role, u.created_at, u.last_login_at, COUNT(a.id) as account_count
		FROM users u
		LEFT JOIN totp_accounts a ON u.id = a.user_id
		GROUP BY u.id
		ORDER BY u.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserAdminView
	for rows.Next() {
		var u UserAdminView
		var lastLogin sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &lastLogin, &u.AccountCount); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			u.LastLoginAt = &lastLogin.Time
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func deleteUser(db *sql.DB, userID int64) error {
	// Foreign key cascade will automatically delete all totp_accounts
	res, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

type Account struct {
	ID              int64     `json:"id"`
	Issuer          string    `json:"issuer"`
	AccountName     string    `json:"account_name"`
	Category        string    `json:"category"`
	EncryptedSecret string    `json:"-"`
	Digits          int       `json:"digits"`
	Period          int       `json:"period"`
	Algorithm       string    `json:"algorithm"`
	CreatedAt       time.Time `json:"created_at"`
}

func listAccounts(db *sql.DB, userID int64) ([]Account, error) {
	rows, err := db.Query(`SELECT id, issuer, account_name, COALESCE(category, 'Personal'), encrypted_secret, digits, period, algorithm, created_at
		FROM totp_accounts WHERE user_id = ? ORDER BY issuer COLLATE NOCASE, account_name COLLATE NOCASE`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Issuer, &a.AccountName, &a.Category, &a.EncryptedSecret, &a.Digits, &a.Period, &a.Algorithm, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func listCategories(db *sql.DB, userID int64) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT COALESCE(category, 'Personal') FROM totp_accounts WHERE user_id = ? ORDER BY category ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil && cat != "" {
			categories = append(categories, cat)
		}
	}
	return categories, rows.Err()
}

func getAccount(db *sql.DB, userID, accountID int64) (*Account, error) {
	row := db.QueryRow(`SELECT id, issuer, account_name, COALESCE(category, 'Personal'), encrypted_secret, digits, period, algorithm, created_at
		FROM totp_accounts WHERE id = ? AND user_id = ?`, accountID, userID)
	var a Account
	if err := row.Scan(&a.ID, &a.Issuer, &a.AccountName, &a.Category, &a.EncryptedSecret, &a.Digits, &a.Period, &a.Algorithm, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func insertAccount(db *sql.DB, userID int64, issuer, accountName, category, encryptedSecret string, digits, period int, algorithm string) error {
	if category == "" {
		category = "Personal"
	}
	_, err := db.Exec(`INSERT INTO totp_accounts (user_id, issuer, account_name, category, encrypted_secret, digits, period, algorithm, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, issuer, accountName, category, encryptedSecret, digits, period, algorithm, time.Now().UTC())
	return err
}

func deleteAccount(db *sql.DB, userID, accountID int64) error {
	_, err := db.Exec(`DELETE FROM totp_accounts WHERE id = ? AND user_id = ?`, accountID, userID)
	return err
}

func getSystemStats(db *sql.DB) (totalUsers int, totalAccounts int, err error) {
	err = db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if err != nil {
		return 0, 0, err
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM totp_accounts`).Scan(&totalAccounts)
	if err != nil {
		return totalUsers, 0, err
	}
	return totalUsers, totalAccounts, nil
}

// --- Brute-force login protection ---

func loginFailures(db *sql.DB, ip string) (failures int, lockedUntil *time.Time, err error) {
	row := db.QueryRow(`SELECT failures, locked_until FROM login_attempts WHERE ip = ?`, ip)
	var lu sql.NullTime
	err = row.Scan(&failures, &lu)
	if err == sql.ErrNoRows {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	if lu.Valid {
		lockedUntil = &lu.Time
	}
	return failures, lockedUntil, nil
}

func recordLoginFailure(db *sql.DB, ip string) error {
	_, err := db.Exec(`
		INSERT INTO login_attempts (ip, failures, locked_until) VALUES (?, 1, NULL)
		ON CONFLICT(ip) DO UPDATE SET
			failures = failures + 1,
			locked_until = CASE WHEN failures + 1 >= 5 THEN datetime('now', '+5 minutes') ELSE locked_until END
	`, ip)
	return err
}

func clearLoginFailures(db *sql.DB, ip string) error {
	_, err := db.Exec(`DELETE FROM login_attempts WHERE ip = ?`, ip)
	return err
}

// --- User Session Management ---

type UserSession struct {
	ID         string     `json:"id"`
	UserID     int64      `json:"userId"`
	IPAddress  string     `json:"ipAddress"`
	UserAgent  string     `json:"userAgent"`
	DeviceName string     `json:"deviceName"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	IsCurrent  bool       `json:"isCurrent,omitempty"`
}

func createUserSession(db *sql.DB, id string, userID int64, ip, userAgent, deviceName string) error {
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO user_sessions (id, user_id, ip_address, user_agent, device_name, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			ip_address = excluded.ip_address,
			user_agent = excluded.user_agent,
			device_name = excluded.device_name
	`, id, userID, ip, userAgent, deviceName, now, now)
	return err
}

func isSessionValid(db *sql.DB, sessionID string, userID int64, ip, userAgent, deviceName string) (bool, error) {
	if sessionID == "" || userID == 0 {
		return false, nil
	}
	var revokedAt sql.NullTime
	err := db.QueryRow(`
		SELECT revoked_at FROM user_sessions
		WHERE id = ? AND user_id = ?
	`, sessionID, userID).Scan(&revokedAt)

	if err == sql.ErrNoRows {
		// Auto-register legacy/mock session
		_ = createUserSession(db, sessionID, userID, ip, userAgent, deviceName)
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return !revokedAt.Valid, nil
}

func touchUserSession(db *sql.DB, sessionID string, ip string) error {
	_, err := db.Exec(`
		UPDATE user_sessions
		SET last_seen_at = ?, ip_address = ?
		WHERE id = ? AND revoked_at IS NULL
	`, time.Now(), ip, sessionID)
	return err
}

func getUserActiveSessions(db *sql.DB, userID int64) ([]UserSession, error) {
	rows, err := db.Query(`
		SELECT id, user_id, ip_address, user_agent, device_name, created_at, last_seen_at
		FROM user_sessions
		WHERE user_id = ? AND revoked_at IS NULL
		ORDER BY last_seen_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []UserSession
	for rows.Next() {
		var s UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.DeviceName, &s.CreatedAt, &s.LastSeenAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func revokeUserSession(db *sql.DB, sessionID string, userID int64) error {
	_, err := db.Exec(`
		UPDATE user_sessions
		SET revoked_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, time.Now(), sessionID, userID)
	return err
}

func revokeOtherUserSessions(db *sql.DB, currentSessionID string, userID int64) error {
	_, err := db.Exec(`
		UPDATE user_sessions
		SET revoked_at = ?
		WHERE user_id = ? AND id != ? AND revoked_at IS NULL
	`, time.Now(), userID, currentSessionID)
	return err
}

func revokeAllUserSessions(db *sql.DB, userID int64) error {
	_, err := db.Exec(`
		UPDATE user_sessions
		SET revoked_at = ?
		WHERE user_id = ? AND revoked_at IS NULL
	`, time.Now(), userID)
	return err
}
