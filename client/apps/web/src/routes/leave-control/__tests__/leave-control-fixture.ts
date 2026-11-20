import type { LeaveControl } from "@trenova/shared/types/worker-leave";

/**
 * Leave settings as the `leaveControl` query sends them for a new organization: the server's
 * statutory defaults (rolling twelve months looking back, twelve weeks, twenty-six for a
 * military caregiver, a forty-hour week, twelve months and 1,250 hours to qualify, fifteen
 * days to return a certification), with its decimals as the strings the `Decimal` scalar
 * carries.
 */
export const savedLeaveControl: LeaveControl = {
  id: "lctl_01",
  version: 4,
  measurementMethod: "RollingBackward",
  entitlementWeeks: "12",
  militaryCaregiverWeeks: "26",
  workweekHours: "40",
  eligibilityMonths: 12,
  eligibilityHours: 1250,
  certificationDueDays: 15,
};
