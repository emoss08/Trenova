---
path: /payroll/pay-profiles
aliases: [driver pay rates, pay package, pay plan, mileage rate, cents per mile, percentage pay, driver pay setup]
related:
  - /payroll/pay-codes
  - /payroll/pay-events
  - /payroll/workspace
  - /hr/workers
---

## What it's for
Pay profiles are reusable driver pay packages. A profile holds the pay components a driver earns on each completed move (for example linehaul per mile with optional mileage bands, a percent of revenue, stop pay or detention), plus an optional guaranteed minimum per period and a per diem daily cap. Payroll managers set them up once and assign them to drivers, adding driver-specific rate overrides where one driver's rate differs.

The table shows each profile's **Status**, **Name**, **Type**, **Pay components**, **Guarantee** and number of **Drivers**.

## Tasks

### Create a pay profile
Keywords: new pay package, set up driver pay, add pay rates
1. Open [Pay profiles](/payroll/pay-profiles).
2. Select **New pay profile**.
3. Choose the **Status** and **Classification** (it decides W-2 or 1099 treatment and the GL expense account), and enter a **Name** and **Description**.
4. Optionally set a **Guaranteed minimum / period** and a **Per diem daily cap**.
5. Under **Pay components**, select **Add component**, then choose the **Component** and **Method** and enter the rate. For a percent-of-revenue method, also choose the **Revenue basis**. Optionally set a **Label**, **Minimum per move** and **Maximum per move**.
6. For per-mile pay, select **Add band** to add **Mileage bands** that pay a different rate by length of haul.
7. Select **Save**.

### Edit a pay profile
Keywords: change pay rate, update pay package
1. Open [Pay profiles](/payroll/pay-profiles) and select the profile.
2. On the **Profile** tab, change the fields or components, then select **Save**.

### Assign a pay profile to a driver
Keywords: assign driver pay, driver rate override, team split
1. Open [Pay profiles](/payroll/pay-profiles) and select the profile.
2. Open the **Assigned drivers** tab and select **Assign driver**.
3. Choose the **Driver**, the **Effective From** date and the **Split percent** (100 for a solo driver, 50 each for an even team split), and add **Notes** if needed.
4. Under **Driver-specific rate overrides**, enter any rate that differs for this driver, then select **Assign profile**.

### Activate or deactivate profiles
Keywords: retire pay profile, disable pay profile
1. Open [Pay profiles](/payroll/pay-profiles) and tick the profiles.
2. Select **Update status**, then **Activate** or **Deactivate**.

## Notes
Needs read access to driver pay profiles to open the page; creating and editing need the matching permissions. The page is only available when the organization has the Asset operations capability.

A deactivated profile can no longer be assigned, but drivers already on it keep being paid under it. Drivers can also be assigned from the Pay tab on the worker in [Workers](/hr/workers).
