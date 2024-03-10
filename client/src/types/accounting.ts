/*
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

import { StatusChoiceProps } from "@/types/index";
import { BaseModel } from "@/types/organization";
import {
  AccountClassificationChoiceProps,
  AccountSubTypeChoiceProps,
  AccountTypeChoiceProps,
  AutomaticJournalEntryChoiceType,
  CashFlowTypeChoiceProps,
  ThresholdActionChoiceType,
} from "@/lib/choices";

/** Types for Division Codes */
export interface DivisionCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  apAccount?: string | null;
  cashAccount?: string | null;
  expenseAccount?: string | null;
}

export interface Tag extends BaseModel {
  id: string;
  name: string;
  description?: string | null;
}

export type TagFormValues = Omit<
  Tag,
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;

export type DivisionCodeFormValues = Omit<
  DivisionCode,
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;

/** Types for General Ledger Accounts */
export interface GeneralLedgerAccount extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  accountNumber: string;
  accountType: AccountTypeChoiceProps | "";
  cashFlowType?: CashFlowTypeChoiceProps | "" | null;
  accountSubType?: AccountSubTypeChoiceProps | "" | null;
  accountClassification?: AccountClassificationChoiceProps | "" | null;
  balance: number;
  openingBalance: number;
  closingBalance: number;
  parentAccount?: string | null;
  isReconciled?: boolean;
  dateOpened?: Date | null;
  dateClosed?: Date | null;
  notes?: string | null;
  owner?: string | null;
  isTaxRelevant?: boolean;
  attachment?: any | null;
  interestRate?: number | null;
  tags?: string[] | null | "";
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
>;

/** Types for Revenue Codes */
export interface RevenueCode extends BaseModel {
  id: string;
  code: string;
  description: string;
  expenseAccount?: string;
  revenueAccount?: string;
  revAccountNum?: string;
  expAccountNum?: string;
}

export type RevenueCodeFormValues = Omit<
  RevenueCode,
  | "id"
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "revAccountNum"
  | "expAccountNum"
>;

/** Types for Accounting Control */
export interface AccountingControl extends BaseModel {
  id: string;
  organizationId: string;
  autoCreateJournalEntries: boolean;
  journalEntryCriteria: AutomaticJournalEntryChoiceType;
  restrictManualJournalEntries: boolean;
  requireJournalEntryApproval: boolean;
  defaultRevenueAccountId?: string | null;
  defaultExpenseAccountId?: string | null;
  enableRecNotifications: boolean;
  reconciliationNotificationRecipients?: string[] | null;
  recThreshold: number;
  recThresholdAction: ThresholdActionChoiceType;
  haltOnPendingRec: boolean;
  criticalProcesses?: string | null;
}

export type AccountingControlFormValues = Omit<
  AccountingControl,
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;
