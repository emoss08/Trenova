---
path: /admin/organization-settings
aliases: [company settings, company profile, organization profile, SSO, single sign-on, SCIM, logo, DOT number, SCAC, security briefing, two-factor coverage, MFA, session timeout]
related:
  - /admin/users
  - /admin/roles
  - /admin/integrations
  - /admin/audit-logs
  - /organization/data-retention
---

## What it's for
Organization settings holds the organization's profile and sign-in security, in two tabs:
- **General**: the logo (**Organization branding**), **Organization details** (name, timezone and
  tenant login slug), the **Operating model**, **Regulatory compliance** identifiers (SCAC, DOT
  number and tax ID) and the **Registered address**.
- **Security**: five sub-tabs. **Overview** is the security briefing: what needs attention, the
  members without two-factor, failed sign-ins, API key hygiene, recent activity and the server's
  session limits.
  **Sign-in** manages OIDC identity providers for single sign-on, **Provisioning** manages SCIM
  directories, tokens and group-to-role mappings, **Policies** holds the **Access settings** and
  priority-ordered access policies, and **Activity** shows sign-ins, risk decisions, external
  identities and two-factor devices.

Administrators use it to keep the company's details current and to set up single sign-on and user
provisioning.

## Tasks

### Update the company profile
Keywords: company name, timezone, DOT number, SCAC, tax ID, address
1. Open [Organization settings](/admin/organization-settings) on the **General** tab.
2. Edit **Organization details** (**Name**, **Timezone**, **Tenant login slug**), **Regulatory
   compliance** (**SCAC code**, **DOT number**, **Tax ID**) and the **Registered address**.
3. Select **Save changes**.

### Change the organization logo
Keywords: upload logo, company logo, branding
1. Open [Organization settings](/admin/organization-settings) on the **General** tab.
2. In **Organization branding**, select **Upload logo** and choose an image.
3. Adjust the crop and select **Upload logo**. Use **Remove logo** to take it off.

### Set the operating model
Keywords: brokerage, asset carrier, hybrid, hide brokerage menus
1. Open [Organization settings](/admin/organization-settings) on the **General** tab.
2. In **Operating model**, pick a **Preset**: Asset, Brokerage or Hybrid. The preset sets the
   **Brokerage features** and **Asset operations** switches, which can also be changed one at a
   time.
3. Select **Save changes**.

### Review the organization's security
Keywords: security briefing, MFA, two-factor, dormant accounts, failed sign-ins, API keys, session timeout, idle sign-out
1. Open [Organization settings](/admin/organization-settings) and select the **Security** tab. It
   opens on **Overview**.
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
1. Open [Organization settings](/admin/organization-settings) and select the **Security** tab, then
   **Sign-in**.
2. Select **New provider**. Under **Quick start**, pick a provider such as Entra ID or Okta to fill
   in common defaults.
3. Fill in the **OIDC application** values: **Issuer URL**, **Redirect URI**, **Client ID**,
   **Client secret** and **OIDC scopes**.
4. Under **Access boundaries**, enter **Allowed domains** and turn on **Enabled**, and optionally
   **Enforce SSO**, **Auto-provision users** and **Trust federated MFA**.
5. Select **Save**.

### Provision users from a directory with SCIM
Keywords: SCIM token, directory sync, group mapping, user provisioning
1. Open [Organization settings](/admin/organization-settings) and select the **Security** tab, then
   **Provisioning**.
2. Under **SCIM directories**, select **New directory**, enter the **Tenant slug**, turn on **Enabled** and
   select **Save**.
3. With the directory selected, enter a token name under **SCIM tokens** and select **New
   token**. Copy the token straight away; it is only shown once.
4. Under **Group role mappings**, select **New mapping** and map an **External group ID** to a
   **Role**, then select **Save**.

## Notes
Opening the page needs read access to the organization; saving the **General** tab needs update
access. Each part of the security briefing appears only to people who can read what it is drawn
from: member figures need users, two-factor coverage needs MFA authenticators, failed sign-ins need
auth events and API key figures need API keys. The operating model only changes what appears in menus and navigation; it does not change
permissions, API access or existing records. Revoke a SCIM token with **Revoke** in its row once it
is no longer used.
