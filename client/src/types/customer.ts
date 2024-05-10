import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "./organization";

/** Customer Type */
export interface Customer extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  name: string;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  zipCode?: string;
  state?: string;
  hasCustomerPortal?: boolean;
  autoMarkReadyToBill?: boolean;
  created: string;
  modified: string;
  advocate?: string;
  advocateFullName?: string;
  lastBillDate?: string;
  lastShipDate?: string;
  totalShipments?: number;
  deliverySlots?: DeliverySlot[];
  contacts?: CustomerContact[];
  emailProfile?: CustomerEmailProfile;
  ruleProfile?: CustomerRuleProfile;
}

export type CustomerFormValues = Omit<
  Customer,
  | "id"
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "advocateFullName"
  | "lastBillDate"
  | "lastShipDate"
  | "deliverySlots"
  | "emailProfile"
  | "ruleProfile"
  | "contacts"
  | "totalShipments"
> & {
  deliverySlots?: DeliverySlotFormValues[] | null;
  contacts?: CustomerContactFormValues[] | null;
  emailProfile: CustomerEmailProfileFormValues;
  ruleProfile: CustomerRuleProfileFormValues;
};

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
  subject?: string;
  comment?: string;
  customer: string;
  fromAddress?: string;
  blindCopy?: string;
  readReceipt: boolean;
  readReceiptTo?: string;
  attachmentName?: string;
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
  locationName: string;
  created: string;
  modified: string;
};

export type DeliverySlotFormValues = Omit<
  DeliverySlot,
  | "id"
  | "organization"
  | "businessUnit"
  | "customer"
  | "locationName"
  | "created"
  | "modified"
>;

type CustomerContact = {
  id: string;
  organization: string;
  businessUnit: string;
  customer: string;
  status: StatusChoiceProps;
  name: string;
  email?: string;
  title?: string;
  phone?: string;
  isPayableContact: boolean;
  created: string;
  modified: string;
};

export type CustomerContactFormValues = Omit<
  CustomerContact,
  "id" | "organization" | "businessUnit" | "customer" | "created" | "modified"
>;
