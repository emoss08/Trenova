---
path: /admin/api-keys
aliases: [API tokens, bearer tokens, access tokens, developer keys, machine credentials, integration keys, service accounts]
related:
  - /admin/integrations
  - /admin/graphql-explorer
  - /admin/audit-logs
---

## What it's for
API keys are bearer credentials that let a third-party system (a warehouse connector, a customer
portal, a reporting tool) call Trenova's API directly. Each key has its own permissions, granted
resource by resource, and is scoped to your organization. Figures at the top show **Total keys**,
**Active keys**, **Revoked keys** and **Requests (30d)**.

The table lists each key's **Name**, **Description**, **Status**, **Permissions**, **Last used**,
**Expires** and **Updated**. Administrators and developers create keys here, hand the token to the
system that needs it, and revoke keys that are no longer used.

## Tasks

### Create an API key
Keywords: new API key, generate token, issue credential, bearer token
1. Open [API keys](/admin/api-keys).
2. Select **New API key**.
3. Under **Key details**, enter a **Display name** and optional **Description**, and set an
   **Expiration** if the key should stop working on a date (leave it blank to keep it active until
   revoked).
4. Under **Permissions**, apply a preset (**All read**, **All write** or **Full access**) or use
   **Search resources...** and grant each resource **View only** or **Full access**. **Clear all**
   removes every grant.
5. Select **Create API key**.
6. Select **Copy API key** and store the token securely, then **Close**. The token is shown only
   once.

### Change a key's details or permissions
Keywords: edit API key, narrow permissions, extend expiration
1. Open [API keys](/admin/api-keys) and select the key's row.
2. Change the **Display name**, **Description**, **Expiration** or permissions.
3. Select **Save changes**.

### Rotate a key's secret
Keywords: regenerate token, new secret, leaked key
1. Open [API keys](/admin/api-keys) and select the key's row.
2. Select the rotate button at the top of the panel (**Rotate secret**). It runs straight away.
3. Select **Copy API key**, update the system that uses the key, and **Close**. The new token is
   shown only once.

### Revoke a key
Keywords: disable API key, delete token, cut off access
1. Open [API keys](/admin/api-keys).
2. Right-click the key and choose **Revoke** (or open it and select the revoke button, **Revoke
   key**).
3. Confirm with **Revoke key**. Any system using the key starts failing authentication
   immediately. A revoked key stays in the list, read-only.

## Notes
Viewing the page needs read access to API keys. Creating a key needs create access to API keys,
and editing, rotating or revoking one needs update access. Requests made with a key are limited to
the permissions granted to that key.
