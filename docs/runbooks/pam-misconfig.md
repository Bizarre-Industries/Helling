# Runbook: PAM Misconfiguration

## Trigger

- Login failures for valid credentials.
- Auth service diagnostics indicate PAM initialization/auth errors.

## Procedure

1. Verify `/etc/pam.d/helling` exists and is readable.
2. Validate `auth.pam_service` matches configured PAM service name.
3. Test PAM stack behavior with host-native authentication tools.
4. Review recent PAM/SSSD package or config changes.
5. Restore known-good PAM config if regression is confirmed.
6. Restart `hellingd` after correction.

## Validation

- Successful `POST /api/v1/auth/login` with valid test account.
- Failed-login rate limiting still enforced.
- No sensitive credential leakage in logs.

## Rollback

- Restore previous PAM config from backup.
- Revert related auth config changes and retest.
