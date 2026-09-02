# 🦦 Ottie - Cute & Secure Self-Hosted TOTP Manager

Ottie is a lightweight, single-binary, self-hosted 2FA authenticator app written in Go with SQLite. It stashes your TOTP secrets securely encrypted in its den and surfaces live 6-digit codes over a playful, intuitive, and cute green-themed web UI.



---

## ✨ Features

- **🦦 Cute & Intuitive Green Aesthetic**: Sage, mint, and deep forest emerald palette with an animated otter mascot, circular countdown timers, and smooth micro-interactions.
- **🔒 True Zero-Knowledge Per-User Encryption**:
  - Each account derives a Key Encryption Key (KEK) from its own master password using **Argon2id** (`64MB`, `3 iterations`).
  - Secrets are encrypted at rest with a random 32-byte Data Encryption Key (DEK) using **AES-256-GCM**.
  - **Admins cannot view or decrypt users' TOTP secrets**. Each user is entirely responsible for their own data.
- **🚀 Seamless On-Ramp Setup Wizard (`/setup`)**:
  - Fresh installations automatically redirect to a friendly 3-step setup wizard to create the root administrator account.
- **🛡️ Multi-User Admin Management (`/admin`)**:
  - Administrators can create user accounts (setting initial roles and passwords).
  - Administrators can delete user accounts, triggering an automatic cascading purge of all associated encrypted records.
  - Zero-knowledge data isolation ensures admins only see token counts, never plaintext secrets or issuers.
- **⚡ 1-Click Fast Copy & Live Search**:
  - Click any code or copy button to instantly copy to clipboard with a mascot toast notification.
  - Filter your vault in real-time by typing issuer or account names.
- **📷 QR Code Scanner & Manual Import**:
  - Drag and drop QR code screenshots or paste `otpauth://` URIs / base32 raw secrets.
  - Live confirmation code verification catches typos before saving.
- **📦 Single Static Binary**:
  - HTML templates and static assets are embedded with `//go:embed`.
  - Built with `modernc.org/sqlite` (pure Go SQLite, zero CGO required).
- **💾 Vault Backup Export**:
  - Users can export a decrypted JSON backup of their tokens and `otpauth://` URIs anytime.

---

## 🚀 Quick Start (Local)

```bash
# 1. Clone & Run
git clone https://github.com/your-username/ottie.git
cd ottie
go run .
```

Visit **`http://localhost:8080`** in your browser:
1. Ottie will greet you with the **First-Time Setup On-Ramp** (`/setup`).
2. Enter your root admin username and password.
3. Start stashing your 2FA accounts!

---

## 🐳 Quick Start (Docker)

```bash
cp .env.example .env
docker compose up -d --build
```

Access the UI at `http://localhost:8080`. The SQLite database is safely persisted in the `ottie_data` Docker volume.

---

## 🔐 Security & Zero-Knowledge Architecture

```
User Password ──(Argon2id)──► User KEK (Key Encryption Key)
                                 │
                 ┌───────────────┴───────────────┐
                 ▼                               ▼
       Wrapped DEK at rest            User Active Session DEK
     (Stored in SQLite users)            (In-memory only)
                                                 │
                                                 ▼
                                     AES-256-GCM Decryption
                                                 │
                                                 ▼
                                       Live 6-Digit TOTP Code
```

- **Data Privacy**: If the server or SQLite database file (`ottie.db`) is leaked or inspected, all TOTP secrets remain encrypted with individual user DEKs that require the respective user's password to unwrap.
- **Admin Isolation**: Admins can manage account lifecycles (create, list, delete) but cannot read or decrypt any user's TOTP seeds.
- **Brute-Force Lockout**: IP-based rate limiting blocks repeated failed login attempts.
- **CSRF Protection**: All mutating forms require single-use CSRF tokens.

---

## 🛠️ Testing & Verification

Run the full automated test suite:

```bash
go test -v ./...
```

Build the standalone binary:

```bash
go build -trimpath -ldflags="-s -w" -o ottie .
```

---

Designed with 💚 for security and simplicity.
