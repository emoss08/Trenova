import type {
  AccountClassificationChoiceProps,
  AccountSubTypeChoiceProps,
  AccountTypeChoiceProps,
  AutomaticJournalEntryChoiceType,
  CashFlowTypeChoiceProps,
  ThresholdActionChoiceType,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";

/** Types for Division Codes */
export interface DivisionCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  apAccountId?: string | null;
  cashAccountId?: string | null;
  expenseAccountId?: string | null;
}

export interface Tag extends BaseModel {
  id: string;
  name: string;
  description?: string | null;
}

export type TagFormValues = Pick<Tag, "id">;

export type DivisionCodeFormValues = Omit<
  DivisionCode,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

/** Types for General Ledger Accounts */
export interface GeneralLedgerAccount extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  accountNumber: string;
  accountType: AccountTypeChoiceProps;
  cashFlowType?: CashFlowTypeChoiceProps | null;
  accountSubType?: AccountSubTypeChoiceProps | null;
  accountClassification?: AccountClassificationChoiceProps | null;
  balance: number;
  openingBalance: number;
  closingBalance: number;
  isReconciled?: boolean;
  dateOpened?: string | null;
  notes?: string;
  isTaxRelevant?: boolean;
  interestRate?: number | null;
  tagIds?: string[];
  tags?: TagFormValues[];
}

export type GLAccountFormValues = Omit<
  GeneralLedgerAccount,
  | "id"
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "dateOpened"
  | "dateClosed"
  | "openingBalance"
  | "closingBalance"
  | "balance"
  | "version"
>;

/** Types for Revenue Codes */
export interface RevenueCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  expenseAccountId?: string | null;
  revenueAccountId?: string | null;
  expenseAccount?: GeneralLedgerAccount | null;
  revenueAccount?: GeneralLedgerAccount | null;
  color?: string;
}

export type RevenueCodeFormValues = Omit<
  RevenueCode,
  | "id"
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "expenseAccount"
  | "revenueAccount"
  | "version"
>;

/** Types for Accounting Control */
export interface AccountingControl extends BaseModel {
  id: string;
  organizationId: string;
  autoCreateJournalEntries: boolean;
  journalEntryCriteria: AutomaticJournalEntryChoiceType;
  restrictManualJournalEntries: boolean;
  requireJournalEntryApproval: boolean;
  defaultRevAccountId?: string | null;
  defaultExpAccountId?: string | null;
  enableRecNotifications: boolean;
  reconciliationNotificationRecipients?: string[] | null;
  recThreshold: number;
  recThresholdAction: ThresholdActionChoiceType;
  haltOnPendingRec: boolean;
  criticalProcesses?: string;
}

export type AccountingControlFormValues = Omit<
  AccountingControl,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;
