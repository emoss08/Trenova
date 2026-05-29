# Encryption and Cloud KMS

This document explains what Trenova encrypts and what is managed in Google Cloud KMS.

## Plain-language summary

Trenova encrypts sensitive customer data before storing it. Google Cloud KMS manages the master key that allows Trenova to unlock the smaller encryption keys used for that data.

Google Cloud KMS does not store, manage, or process customer documents, uploaded files, API keys, or business records. It only protects the keys needed to decrypt them.

## What is encrypted

Trenova encrypts:

- Uploaded document file contents
- Document preview content
- Integration API keys and tokens
- SSO and OIDC client secrets
- IAM OIDC client secrets
- EDI communication and partner secrets

The encrypted data is stored in Trenova-controlled storage systems, such as object storage and the application database. The original plaintext content is not stored.

## What Google Cloud KMS manages

Google Cloud KMS manages the master encryption key used to protect Trenova's per-item encryption keys.

KMS is responsible for:

- Holding the master key used to wrap and unwrap data encryption keys
- Enforcing Google Cloud IAM access to that key
- Recording audit logs when the key is used
- Managing key versions
- Supporting key rotation

KMS does not manage:

- Uploaded documents
- Document previews
- API keys or secret values
- Customer shipment, billing, or business data
- File storage permissions
- User permissions inside Trenova

## How document encryption works

When a user uploads a document:

1. Trenova creates a unique encryption key for that document.
2. Trenova encrypts the document content before storing it.
3. Trenova asks Google Cloud KMS to protect the document's encryption key.
4. Trenova stores the encrypted document and the protected key metadata.

When a user downloads or previews a document:

1. Trenova checks that the user is allowed to access the document.
2. Trenova asks Google Cloud KMS to unlock the protected document key.
3. Trenova decrypts the document for the authorized request.
4. The decrypted content is streamed through Trenova's API.

Google Cloud KMS never receives the document content. It only receives the small encryption key needed to unlock that document.

## Why this matters

This design separates stored data from the keys needed to read it.

If object storage were accessed directly, document contents would still be encrypted. If database metadata were accessed directly, the protected encryption keys would still require Google Cloud KMS access. Production access to the master key is controlled and audited through Google Cloud IAM and Cloud KMS.

## Key rotation

Google Cloud KMS supports rotating the master key over time. When rotation happens, Trenova can re-protect stored document and secret encryption keys with the active KMS key version without changing the original document contents.

This means key rotation does not require users to re-upload documents or re-enter secrets.

## What users should expect

Users continue to upload, preview, and download documents through Trenova. The encryption process is automatic.

Users should not upload documents directly to object storage or bypass Trenova's API. Trenova's API is responsible for authorization checks, encryption, decryption, and auditability.
