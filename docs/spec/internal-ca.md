# Internal CA Specification

This document specifies the Certificate Authority (CA) lifecycle for Helling's per-user Incus client certificates.

## Overview

Helling maintains an internal CA to issue per-user client certificates for mTLS authentication with Incus. Each user receives a unique certificate that grants them access to their own Incus resources via the proxy.

## CA Key Management

### Key Type

- **Algorithm:** Ed25519 (RFC 8037 per ADR-031)
- **Location:** `/etc/helling/ca.key.age` (file, encrypted with age per ADR-039)
- **Permissions:** `0400` (readable by `helling` user only)
- **Ownership:** `helling:helling`

### CA Key Encryption

- **Method:** age (filippo.io/age v1.2.x per ADR-039)
- **Identity:** Stored in `/var/lib/helling/ca-identity` (file, not encrypted, created at first boot)
- **Derivation:** Identity derived from host entropy (e.g., machine-id + random seed)
- **Rotation:** Identity is permanent for a host; CA key rekeying requires manual intervention

### Bootstrap Process

1. At first hellingd startup:
   - Check if `/var/lib/helling/ca-identity` exists
   - If not, generate new age identity → store to file
   - Generate CA key (Ed25519) → encrypt with age identity → write to `/etc/helling/ca.key.age`
   - Record CA cert validity window in SQLite

2. On each hellingd startup:
   - Read age identity from `/var/lib/helling/ca-identity`
   - Decrypt `/etc/helling/ca.key.age` using identity
   - Load into memory (runtime use only)
   - **Never** store unencrypted key to disk

## CA Certificate

### Validity Period

- **Validity:** 5 years from generation date
- **Storage:** `/etc/helling/ca.crt` (plaintext PEM, no secrets)
- **Subject:** `CN=Helling CA, O=Bizarre Industries, C=US` (or configurable)
- **Public key:** Ed25519 (matches CA key)

### Issuance

CA certificate is self-signed and issued once at CA key generation:

```bash
openssl genpkey -algorithm ED25519 -out ca.key
openssl req -new -x509 -key ca.key -days 1825 -out ca.crt \
  -subj "/CN=Helling CA/O=Bizarre Industries"
```

(Or equivalent in Go using `crypto/x509` and `crypto/ed25519`.)

## User Certificates

### Validity Period

- **Validity:** 90 days from issuance
- **Auto-renewal:** Triggered at 60 days remaining validity
- **Grace period:** 10 days after expiry before user certificate becomes invalid

### Certificate Lifecycle

**Issuance (during user creation):**

1. Hellingd generates Ed25519 key pair for user
2. Hellingd creates Certificate Signing Request (CSR) with:
   - Subject: `CN={username}, O=Helling Users, C=US`
   - Constraints: Key Usage = `digitalSignature`, Extended Key Usage = `clientAuth`
3. Hellingd self-signs CSR using CA key → user certificate
4. Both user private key + certificate stored encrypted in SQLite (separate fields, both with age)

**Renewal (automatic at 60 days remaining):**

1. Hellingd queries SQLite for all user certs with ≤60 days validity
2. For each user:
   - Generate new key pair
   - Create CSR
   - Sign with CA key
   - Store new cert + key encrypted
   - Old cert + key marked `superseded` in SQLite (not deleted)

**Concurrent Validity (Dual-Sign Period):**

- During renewal, both old and new user certs are valid (60-day overlap)
- Incus trusts both; requests using old cert still accepted
- At day 60 after renewal, old cert marked `expired` and no longer usable

**Grace Period (10 days after technical expiry):**

- If user certificate hasn't been renewed by expiry date + 10 days, user is locked out
- Audit log: `user_certificate_revoked` + manual unlock required

### Storage Format

User certificates stored in SQLite table `user_certificates`:

```sql
CREATE TABLE user_certificates (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  serial_number TEXT NOT NULL,
  cert_pem TEXT NOT NULL,      -- encrypted with age
  private_key_pem TEXT NOT NULL, -- encrypted with age
  issued_at TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  status TEXT NOT NULL,          -- active|superseded|expired
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE(user_id, status) -- only one active cert per user
);
```

All PEM text fields are encrypted using age with a per-user key stored in the `users` table.

### Hashing

User certificate hashes (for audit) are stored separately:

```sql
CREATE TABLE user_certificate_hashes (
  cert_serial TEXT PRIMARY KEY REFERENCES user_certificates(serial_number),
  sha256_hash TEXT NOT NULL, -- plaintext hash for display/audit
  created_at TIMESTAMP NOT NULL
);
```

## CA Rotation

### Rationale

CA key rotation is needed to:

- Limit blast radius of CA key compromise
- Refresh cryptographic material after extended use
- Respond to security advisories

### Procedure

1. **Plan:** Schedule rotation window (maintenance window with service downtime acceptable)
2. **Generate:** Create new CA key + certificate (same process as bootstrap)
3. **Dual-Sign:** For 60 days, Incus trusts both old and new CA certs
   - Store both CA certs in Incus trust store (one `ca.crt`, one `ca-old.crt`)
   - All new user certs signed with new CA key
   - Existing user certs signed with old CA key continue to work
4. **Rekey Users:** Re-issue user certificates under new CA (during renewal cycle)
5. **Retire Old:** After 60 days, remove old CA cert from Incus trust store
6. **Archive:** Retain old CA cert + key (encrypted) in secure backup for audit/forensics

### Implementation

CA rotation is manual (not automatic) and requires:

- Admin login to hellingd host
- Running `helling upgrade --rotate-ca` command (creates new CA, triggers audit)
- Waiting for user certificate renewals to complete (automatic, 60 days)

## Certificate Usage

### By Helling (Proxy)

When forwarding a request to Incus as user `alice`:

1. Look up `alice`'s active certificate from SQLite
2. Decrypt private key + certificate using age
3. Build `tls.Config` with:
   - `Certificates: []tls.Certificate{cert}` (user's client cert)
   - `RootCAs` configured to trust Incus's self-signed cert (loopback CA)
4. Create HTTP client with this `tls.Config`
5. Send request to Incus HTTPS loopback API

### By Incus

Incus is configured to trust the Helling CA:

- Incus server certificate is issued by Incus's own CA (self-signed)
- Helling proxy connects via `/1.0` HTTPS (TLS mutual authentication)
- Incus validates user's client certificate against Helling CA
- Incus checks certificate CN to infer user identity (optional; Helling forwards via header)

## Compliance

- **Key Material:** Ed25519 keys stored encrypted at rest, unencrypted in memory during operation
- **Hashing:** User certs not hashed for storage (full PEM encrypted); hashes computed for audit display
- **Audit:** All CA operations logged:
  - `ca_key_generated` — at bootstrap
  - `user_cert_issued` — on user creation
  - `user_cert_renewed` — automatic renewal
  - `user_cert_revoked` — grace period expiry
  - `ca_rotated` — manual CA rotation

## References

- ADR-030 (argon2id password hashing)
- ADR-031 (Ed25519 JWT)
- ADR-039 (age encryption)
- ADR-050 (hellingd non-root user)
- RFC 8037 (Ed25519 in X.509)
- RFC 9106 (Argon2 Password Hashing)
