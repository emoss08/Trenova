---
path: /admin/organization-settings
aliases: [company settings, company profile, organization profile, logo, icon, branding, DOT number, SCAC, tax ID, registered address, timezone, sign-in address, login slug, operating model]
related:
  - /admin/security-access
  - /admin/users
  - /admin/roles
  - /admin/integrations
  - /organization/data-retention
---

## What it's for
Organization settings holds the organization's own record, in three tabs saved with one
**Save changes**:
- **Profile**: the **Name**, **Timezone** and **Sign-in address** (the page your people sign in
  at, such as /login/acme-logistics), and the **Registered address**.
- **Branding**: the **Logo and icon**, changed in a live preview of the sidebar and an invoice.
  A picture is saved as soon as you place it, apart from the form.
- **Operations**: the **Operating model** (brokerage, asset operations or both) and the
  **Regulatory** identifiers: **SCAC code**, **DOT number** and **Tax ID**.

Administrators use it to keep the company's details current. Single sign-on, user provisioning,
access policies and sign-in activity are on [Security & access](/admin/security-access).

## Tasks

### Update the company profile
Keywords: company name, timezone, sign-in address, login slug, address
1. Open [Organization settings](/admin/organization-settings). It opens on **Profile**.
2. Edit **Name**, **Timezone** and **Sign-in address**, and the **Registered address**: **Address
   line 1**, **Suite or unit**, **City**, **State** and **ZIP code**.
3. Select **Save changes**.

### Update the regulatory identifiers
Keywords: DOT number, SCAC, tax ID, USDOT, EIN
1. Open [Organization settings](/admin/organization-settings) and select **Operations**.
2. Under **Regulatory**, edit the **SCAC code** (four characters), the **DOT number** (up to
   eight digits) and the **Tax ID**.
3. Select **Save changes**.

### Change the organization logo or icon
Keywords: upload logo, company logo, branding, icon, sidebar icon, crop logo, replace logo, remove logo, svg logo
1. Open [Organization settings](/admin/organization-settings) and select **Branding**.
2. In **Logo and icon**, the **Preview** shows your icon in the **Sidebar** and your logo at the
   top of an **Invoice**. Switch to the one you want to change.
3. Select the icon in the sidebar, or the logo on the invoice (a dashed outline marks each),
   and choose an image, or drop one onto it. PNG, JPG, WebP and SVG files up to 20 MB are
   accepted. Until you add an icon, the sidebar fits your logo into its square.
4. Drag the picture into place and zoom with the slider or the scroll wheel; the preview beside
   it shows the result as you move it. Select **Upload logo** or **Upload icon**. The picture is
   saved straight away; anything you changed on the other tabs stays to be saved.
5. To swap or take off a picture later, select it in the preview and choose **Replace** or
   **Remove**.

### Set the operating model
Keywords: brokerage, asset carrier, hybrid, hide brokerage menus
1. Open [Organization settings](/admin/organization-settings) and select **Operations**.
2. In **Operating model**, pick a **Preset**: Asset, Brokerage or Hybrid. The preset sets the
   **Brokerage features** and **Asset operations** switches, which can also be changed one at a
   time. At least one must stay on.
3. Select **Save changes**.

## Notes
Opening the page needs read access to the organization; saving needs update access. The
operating model only changes what appears in menus and navigation, and whether drivers can be
assigned; it does not change permissions, API access or existing records. **Brokerage features**
and the **Preset** appear only when your plan includes carrier brokerage. A save the server
refuses opens the tab holding the field to fix. An address of the old **Security** tab opens
[Security & access](/admin/security-access).
