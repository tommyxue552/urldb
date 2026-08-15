# Credential security and rotation

## JWT

`JWT_SECRET` is required at startup and must contain at least 32 random bytes.
It is injected through the deployment secret store or environment, never stored
in the database or committed to source control. The service signs new tokens
only with this value and accepts `JWT_PREVIOUS_SECRET` only for verification.

Rotation procedure:

1. Generate a new 32-byte-or-longer random secret in the deployment secret store.
2. Set it as `JWT_SECRET`, move the former value to `JWT_PREVIOUS_SECRET`, then deploy every instance.
3. Keep the previous value only for the maximum JWT lifetime (currently 30 days) plus clock skew.
4. Remove `JWT_PREVIOUS_SECRET` and deploy again. Emergency revocation omits step 2 and invalidates every session.

## Provider cookies and tokens

The `cks.ck` and `cks.extra` columns contain provider credentials and must be
encrypted at rest before enabling credential ingestion in production. The next
implementation increment uses envelope encryption: a versioned AES-256-GCM
data-encryption key encrypts each value with a fresh nonce and the account ID
and column name as authenticated data; the key-encryption key stays in the
deployment secret manager/KMS. Reads decrypt in the repository boundary, so
providers continue receiving plaintext only in process memory. Queries must use
a separate keyed HMAC fingerprint, never ciphertext equality, because GCM uses
a random nonce.

Credential rotation is provider-specific: preserve the old value only until a
validated replacement is saved, record the rotation metadata, and securely
retire the old encrypted envelope. Automatic provider refresh-token rotation
must use the same repository write path.

## Audit and logging

Every credential create, update, delete, decrypt failure, and automatic token
rotation must create an append-only audit record with actor (or `system`),
account ID, provider, action, outcome, source IP, request ID, and key version.
The record must not contain credential material, authorization headers, request
bodies, or plaintext/ciphertext snippets. Retain audit logs according to the
deployment retention policy and restrict viewing to administrators.

Until the encrypted repository migration is complete, do not log account create
or update request bodies. Existing API access logs must redact `ck`, `extra`,
`token`, `authorization`, `password`, and related fields before persistence.
