# Security Policy

The Ottie team takes the security of your authentication secrets and credentials very seriously. 

## Supported Versions

We provide security updates and patches for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

---

## Security Architecture & Guarantees

Ottie is engineered with a **True Zero-Knowledge** encryption architecture:

1. **Zero-Knowledge Key Derivation**:
   - Master Vault Keys are never stored in plaintext or transmitted across the wire without TLS.
   - Master keys are derived client/server-side using **Argon2id** (`memory: 64MB`, `iterations: 3`, `parallelism: 4`) with cryptographically random salts.
2. **Authenticated Symmetric Encryption**:
   - Every TOTP secret at rest is encrypted with a unique 32-byte Data Encryption Key (DEK) using **AES-256-GCM**.
   - Authenticated tags guarantee detection of any tampering or ciphertext corruption.
3. **Session Hardening**:
   - Authentication sessions are protected by `HttpOnly`, `SameSite=Lax` cookies with strict 7-day TTLs.
   - CSRF headers and rate-limiting protect authentication endpoints from brute-force attacks.
4. **Automated Continuous Verification**:
   - Code is scanned continuously via GitHub CodeQL, `govulncheck`, and `gosec`.

---

## Reporting a Vulnerability

If you discover a security vulnerability within Ottie:

1. **Do NOT open a public GitHub issue**.
2. Submit your report privately via [GitHub Private Vulnerability Reporting](https://github.com/Ottie-Group/Ottie/security/advisories/new) or email security concerns to the maintainers.
3. Please provide:
   - A description of the issue.
   - Steps or proof-of-concept to reproduce the vulnerability.
   - Any suggested mitigations.

We will acknowledge your report within 48 hours and work with you to release a fix before public disclosure.