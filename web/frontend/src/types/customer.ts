/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



import { IChoiceProps, type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "./organization";

type BillingCycleChoices =
  | "PER_SHIPMENT"
  | "QUARTERLY"
  | "MONTHLY"
  | "ANNUALLY";

export enum EnumBillingCycleChoices {
  PER_SHIPMENT = "PER_SHIPMENT",
  QUARTERLY = "QUARTERLY",
  MONTHLY = "MONTHLY",
  ANNUALLY = "ANNUALLY",
}

/** Returns the billing cycle choices as an array of objects */
export const BillingCycleChoices = [
  { value: "PER_SHIPMENT", label: "Per Shipment", color: "#ff75c3" },
  { value: "QUARTERLY", label: "Quarterly", color: "#ff7f50" },
  { value: "MONTHLY", label: "Monthly", color: "#ffa647" },
  { value: "ANNUALLY", label: "Annually", color: "#dc143c" },
] satisfies ReadonlyArray<IChoiceProps<BillingCycleChoices>>;

/** Customer Rule Profile Type */
interface CustomerRuleProfile extends BaseModel {
  customerId: string;
  docClassIds: string[];
  billingCycle: EnumBillingCycleChoices;
}

export type CustomerRuleProfileFormValues = Omit<
  CustomerRuleProfile,
  | "id"
  | "customerId"
  | "businessUnitId"
  | "organizationId"
  | "version"
  | "createdAt"
  | "updatedAt"
>;

type EmailFormatChoices = "HTML" | "PLAIN";

export enum EnumEmailFormatChoices {
  HTML = "HTML",
  PLAIN = "PLAIN",
}

/**
 * Returns yes and no choices as boolean for a select input
 * @returns An array of yes and no choices as boolean.
 */
export const EmailFormatChoices = [
  { value: "HTML", label: "HTML", color: "#9c25eb" },
  { value: "PLAIN", label: "PLAIN", color: "#2563eb" },
] satisfies ReadonlyArray<IChoiceProps<EmailFormatChoices>>;

/** Customer Email Profile Type */
interface CustomerEmailProfile extends BaseModel {
  customerId: string;
  subject?: string;
  emailProfileId?: string | null;
  emailRecipients: string;
  attachmentName?: string;
  emailCcRecipients?: string;
  emailFormat: EnumEmailFormatChoices;
}

export type CustomerEmailProfileFormValues = Omit<
  CustomerEmailProfile,
  | "id"
  | "customerId"
  | "businessUnitId"
  | "organizationId"
  | "version"
  | "createdAt"
  | "updatedAt"
>;

export enum EnumDayOfWeekChoices {
  SUNDAY = "SUNDAY",
  MONDAY = "MONDAY",
  TUESDAY = "TUESDAY",
  WEDNESDAY = "WEDNESDAY",
  THURSDAY = "THURSDAY",
  FRIDAY = "FRIDAY",
  SATURDAY = "SATURDAY",
}

export const DayOfWeekChoices = [
  { value: "SUNDAY", label: "Sunday", color: "#ff75c3" },
  { value: "MONDAY", label: "Monday", color: "#ff7f50" },
  { value: "TUESDAY", label: "Tuesday", color: "#ffa647" },
  { value: "WEDNESDAY", label: "Wednesday", color: "#dc143c" },
  { value: "THURSDAY", label: "Thursday", color: "#4682b4" },
  { value: "FRIDAY", label: "Friday", color: "#6a5acd" },
  { value: "SATURDAY", label: "Saturday", color: "#9c25eb" },
];

interface DeliverySlot extends BaseModel {
  customerId: string;
  locationId: string;
  dayOfWeek: EnumDayOfWeekChoices;
  startTime: string;
  endTime: string;
}

export type DeliverySlotFormValues = Omit<
  DeliverySlot,
  | "id"
  | "customerId"
  | "businessUnitId"
  | "organizationId"
  | "version"
  | "createdAt"
  | "updatedAt"
>;

interface CustomerContact extends BaseModel {
  id: string;
  customerId: string;
  name: string;
  email?: string;
  title?: string;
  phoneNumber?: string;
  isPayableContact: boolean;
}

export type CustomerContactFormValues = Omit<
  CustomerContact,
  | "id"
  | "customerId"
  | "businessUnitId"
  | "organizationId"
  | "version"
  | "createdAt"
  | "updatedAt"
>;

/** Customer Type */
export interface Customer extends BaseModel {
  id: string;
  status: StatusChoiceProps;

  code?: string;
  name: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  postalCode: string;
  stateId: string;
  hasCustomerPortal?: boolean;
  autoMarkReadyToBill?: boolean;
  ruleProfile?: CustomerRuleProfileFormValues;
  emailProfile?: CustomerEmailProfileFormValues;
  deliverySlots?: DeliverySlotFormValues[];
  contacts?: CustomerContactFormValues[];
}

export type CustomerFormValues = Omit<
  Customer,
  "id" | "organizationId" | "version" | "createdAt" | "updatedAt"
>;
