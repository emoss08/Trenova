---
path: /admin/organization-settings
aliases: [company settings, company profile, organization profile, SSO, single sign-on, SCIM, logo, DOT number, SCAC, subscription, plan usage]
related:
  - /admin/users
  - /admin/roles
  - /admin/integrations
  - /admin/audit-logs
  - /organization/data-retention
---

## What it's for
Organization settings holds the organization's profile, sign-in security and subscription details,
in three tabs:
- **General**: the logo (**Organization branding**), **Organization details** (name, timezone and
  tenant login slug), the **Operating model**, **Regulatory compliance** identifiers (SCAC, DOT
  number and tax ID) and the **Registered address**.
- **Security**: four sub-tabs. **Sign-in** manages OIDC identity providers for single sign-on,
  **Provisioning** manages SCIM directories, tokens and group-to-role mappings, **Policies** holds
  priority-ordered access policies, and **Activity** shows authentication events, risk decisions,
  external identities and MFA authenticators.
- **Billing & usage**: the access state, plan, enabled features and metered usage for the period.

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

### Set up single sign-on
Keywords: SSO, OIDC, Okta, Entra ID, Azure AD, enforce SSO
1. Open [Organization settings](/admin/organization-settings) and select the **Security** tab, then
   **Sign-in**.
2. Select **Add provider**. Under **Quick start**, pick a provider such as Entra ID or Okta to fill
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
2. Under **SCIM directories**, select **Add**, enter the **Tenant slug**, turn on **Enabled** and
   select **Save**.
3. With the directory selected, enter a token name under **SCIM tokens** and select **Create
   token**. Copy the token straight away; it is only shown once.
4. Under **Group role mappings**, select **Add mapping** and map an **External group ID** to a
   **Role**, then select **Save**.

### Check plan and usage
Keywords: subscription, billing plan, usage limits, features
1. Open [Organization settings](/admin/organization-settings) and select the **Billing & usage**
   tab.
2. Review the **Access state**, **Plan**, **Usage this period** and **Enabled features**. Select
   **Refresh** to reload usage.

## Notes
Opening the page needs read access to the organization; saving the **General** tab needs update
access. The operating model only changes what appears in menus and navigation; it does not change
permissions, API access or existing records. Revoke a SCIM token with **Revoke** in its row once it
is no longer used.
