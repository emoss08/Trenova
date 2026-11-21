---
path: /admin/security-access
aliases: [security, SSO, single sign-on, OIDC, SCIM, directory sync, provisioning, access policies, security briefing, two-factor coverage, MFA, session timeout, sign-in activity, risk decisions]
related:
  - /admin/organization-settings
  - /admin/users
  - /admin/roles
  - /admin/api-keys
  - /admin/audit-logs
---

## What it's for
Security & access is where the organization's sign-in security is set and watched, in five tabs:
- **Overview** is the security briefing: what needs attention, the members without two-factor,
  failed sign-ins, API key hygiene, recent activity and the server's session limits.
- **Sign-in** manages OIDC identity providers for single sign-on.
- **Provisioning** manages SCIM directories, tokens and group-to-role mappings.
- **Policies** holds the **Access settings** and priority-ordered access policies.
- **Activity** shows sign-ins, risk decisions, external identities and two-factor devices.

Each tab appears only to people who can read what it holds, and the page is listed in the
navigation for anyone who can read one of them.

An access policy narrows what roles allow, and it binds people, agents and API keys alike. It can
ask for proof as well as name who it applies to: a policy that requires two-factor refuses anyone
signed in without it, whether the policy allows or denies, and an agent working for them is held to
the same sign-in. A policy that denies unless two-factor was used steps aside once it was. API keys
and agents working alone have no sign-in to prove anything with, so a two-factor policy does not
stop them: add a policy that denies them by principal type.

## Tasks

### Review the organization's security
Keywords: security briefing, MFA, two-factor, dormant accounts, failed sign-ins, API keys, session timeout, idle sign-out
1. Open [Security & access](/admin/security-access). It opens on **Overview**.
2. Read **Recommendations**, worst first. Each one says what was found and offers the step that
   settles it, such as **Open users**, **Open policies** or **Review API keys**. Checks that hold
   are folded under a line that counts the checks passed.
3. **Without two-factor** lists the members who still sign in with a password alone. Each person
   turns two-factor on from their own security settings.
4. **Idle sign-out**, among the figures, shows how long an idle browser stays signed in and how
   long a session lasts. These come from the server's configuration and apply to every
   organization it hosts, so they are shown here but not changed.
5. Select **Check again** to run the checks once more after a change.

### Set up single sign-on
Keywords: SSO, OIDC, Okta, Entra ID, Azure AD, enforce SSO
1. Open [Security & access](/admin/security-access) and select **Sign-in**.
2. Select **New provider**. Under **Quick start**, pick a provider such as Entra ID or Okta to fill
   in common defaults.
3. Fill in the **OIDC application** values: **Issuer URL**, **Redirect URI**, **Client ID**,
   **Client secret** and **OIDC scopes**.
4. Under **Access boundaries**, enter **Allowed domains** and turn on **Enabled**, and optionally
   **Enforce SSO**, **Auto-provision users** and **Trust federated MFA**.
5. Select **Save**.

### Provision users from a directory with SCIM
Keywords: SCIM token, directory sync, group mapping, user provisioning, Okta, Entra ID, Azure AD, deprovision
1. Open [Security & access](/admin/security-access) and select **Provisioning**.
2. Under **SCIM directories**, select **New directory**, enter the **Tenant slug**, turn on **Enabled** and
   select **Save**.
3. With the directory selected, copy the **SCIM base URL** shown under its name.
4. Enter a token name under **SCIM tokens** and select **New token**. Copy the token straight away;
   it is only shown once.
5. Under **Group role mappings**, select **New mapping** and map an **External group ID** to a
   **Role**, then select **Save**. Enter the group's name as your identity provider shows it, or its
   object ID for Entra ID.
6. In your identity provider, set up SCIM provisioning with the base URL and the token:
   - Okta: in the app's Provisioning settings, choose SCIM 2.0, paste the base URL as the connector
     base URL, use "userName" as the unique identifier, choose HTTP header authentication and paste
     the token. Turn on creating users, updating user attributes and deactivating users, then push
     the groups you mapped.
   - Entra ID: in the enterprise application's Provisioning page, choose automatic provisioning,
     paste the base URL as the tenant URL and the token as the secret token, select Test
     Connection, then assign the users and groups to provision.

## Notes
SCIM provisioning: people your identity provider assigns are added to this organization. Someone who already belongs
to it is linked rather than added again, and someone whose email belongs to another organization
must be added here by an administrator first. Group membership grants the mapped roles within a
few seconds. Unassigning or deactivating a person in your identity provider ends their access to
this organization; if they also belong to other organizations, that access and their profile are
left alone.

**Overview** needs read access to the organization, **Sign-in** to identity providers,
**Provisioning** to SCIM directories or the provisioning audit, **Policies** to access policies,
and **Activity** to any of auth events, risk decisions, external identities or two-factor devices.
Each part of the security briefing appears only to people who can read what it is drawn from:
member figures need users, two-factor coverage needs MFA authenticators, failed sign-ins need
auth events and API key figures need API keys. Revoke a SCIM token with **Revoke** in its row once
it is no longer used. A group mapping grants its role only while the person who saved it may still
grant that role; if they lose that permission or their account, the role is not granted again and
is taken back from each person the next time the directory changes them, until someone who may
grant it saves the mapping again. Roles granted through a group are managed by the directory:
removing one by hand lasts only until the next sync while the group still grants it.
