---
path: /admin/custom-fields
aliases: [custom field definitions, user-defined fields, extra fields, additional fields, custom attributes, UDF]
related:
  - /equipment/trailers
  - /equipment/tractors
  - /hr/workers
  - /billing/configuration-files/customers
  - /dispatch/locations
  - /shipment-management/shipments
---

## What it's for
Custom field definitions add your own fields to Trenova records when the built-in fields are not
enough, for example a lease number on trailers or a badge ID on workers. Custom fields can be
added to trailers, tractors, workers, shipments, customers and locations. Each definition names
the **Resource type** it applies to, its **Field type** (**Text**, **Number**, **Date**,
**Boolean**, **Select** or **Multi-select**), an internal **Name** and the **Label** people see,
and whether it is **Required** and **Active**.

The table lists each definition's **Label**, **Name**, **Resource type**, **Field type**,
**Required**, **Active** and **Created At**. Administrators maintain it; the fields then appear on
the forms for that kind of record.

## Tasks

### Add a custom field
Keywords: new custom field, add field to trailer, extra field on worker, dropdown field
1. Open [Custom field definitions](/admin/custom-fields).
2. Select **New custom field definition**.
3. Choose the **Resource type** and **Field type**.
4. Enter the **Name** (lowercase letters and underscores only) and the **Label** shown to users,
   and optionally a **Description** as help text.
5. Turn on **Required** if users must provide a value, and leave **Active** on so the field is
   visible and usable. Optionally set a **Display order** and a **Color**.
6. For **Select** or **Multi-select**, under **Options** select **Add option** for each choice
   and fill in its **Value**, **Label** and optional **Color**. At least one option is needed.
7. Select **Save**.

### Change a custom field
Keywords: edit custom field, rename label, add dropdown option
1. Open [Custom field definitions](/admin/custom-fields).
2. Select the field's row (or right-click it and choose **Edit**).
3. Change the fields and select **Save**.

### Deactivate or reactivate a custom field
Keywords: hide custom field, disable field, retire field
1. Open [Custom field definitions](/admin/custom-fields).
2. Right-click an active field and choose **Deactivate**. The field is hidden from forms and the
   values already entered are kept.
3. To bring it back, right-click it and choose **Activate**.

### Delete a custom field
Keywords: remove custom field
1. Open [Custom field definitions](/admin/custom-fields).
2. Right-click the field and choose **Delete**, then confirm with **Delete**. This cannot be
   undone.
3. If the field already holds values on any record, it cannot be deleted; the dialog shows how
   many values exist and asks you to deactivate it instead.

## Notes
Viewing the page needs read access to custom field definitions; **New custom field definition**
appears only for people who can create them.
