/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { StatusChoiceProps } from "@/types/index";

/** Customer Type */
export type Customer = {
  id: string;
  organization: string;
  status: StatusChoiceProps;
  code: string;
  name: string;
  addressLine1?: string | null;
  addressLine2?: string | null;
  city?: string | null;
  zipCode?: string | null;
  state?: string | null;
  hasCustomerPortal?: boolean;
  autoMarkReadyToBill?: boolean;
  created: string;
  modified: string;
  advocate?: string | null;
  advocateFullName?: string | null;
  lastBillDate?: string | null;
  lastShipDate?: string | null;
  totalShipments?: number | null;
  deliverySlots?: DeliverySlotFormValues[] | null;
  contacts?: CustomerContactFormValues[] | null;
  emailProfile?: CustomerEmailProfileFormValues | null;
  ruleProfile?: CustomerRuleProfileFormValues | null;
};

export type CustomerFormValues = Omit<
  Customer,
  | "id"
  | "organization"
  | "created"
  | "modified"
  | "advocateFullName"
  | "lastBillDate"
  | "lastShipDate"
  | "totalShipments"
>;

/** Customer Rule Profile Type */
export type CustomerRuleProfile = {
  id: string;
  organization: string;
  businessUnit: string;
  name: string;
  customer: string;
  documentClass: string[];
  created: string;
  modified: string;
};

export type CustomerRuleProfileFormValues = Omit<
  CustomerRuleProfile,
  "id" | "customer" | "businessUnit" | "organization" | "created" | "modified"
>;

/** Customer Email Profile Type */
export type CustomerEmailProfile = {
  id: string;
  organization: string;
  businessUnit: string;
  subject?: string | null;
  comment?: string | null;
  customer: string;
  fromAddress?: string | null;
  blindCopy?: string | null;
  readReceipt: boolean;
  readReceiptTo?: string | null;
  attachmentName?: string | null;
};

export type CustomerEmailProfileFormValues = Omit<
  CustomerEmailProfile,
  "id" | "customer" | "businessUnit" | "organization"
>;

export type DeliverySlot = {
  id: string;
  organization: string;
  businessUnit: string;
  customer: string;
  dayOfWeek: number;
  startTime: string;
  endTime: string;
  location: string;
  created: string;
  modified: string;
};

export type DeliverySlotFormValues = Omit<
  DeliverySlot,
  "id" | "organization" | "businessUnit" | "customer" | "created" | "modified"
>;

type CustomerContact = {
  id: string;
  organization: string;
  businessUnit: string;
  customer: string;
  isActive: boolean;
  name: string;
  email?: string | null;
  title?: string | null;
  phone?: string | null;
  isPayableContact: boolean;
  created: string;
  modified: string;
};

export type CustomerContactFormValues = Omit<
  CustomerContact,
  "id" | "organization" | "businessUnit" | "customer" | "created" | "modified"
>;
