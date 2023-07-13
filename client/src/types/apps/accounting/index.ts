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

import { StatusChoiceProps } from "@/types";
import {
  AccountClassificationChoiceProps,
  AccountSubTypeChoiceProps,
  AccountTypeChoiceProps,
  CashFlowTypeChoiceProps,
} from "@/utils/apps/accounting";

/** Types for Division Codes */
export type DivisionCode = {
  id: string;
  organization: string;
  created: string;
  modified: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  ap_account?: string | null;
  cash_account?: string | null;
  expense_account?: string | null;
};

export interface DivisionCodeFormValues
  extends Omit<DivisionCode, "id" | "organization" | "created" | "modified"> {}

/** Types for General Ledger Accounts */
export type GeneralLedgerAccount = {
  id: string;
  organization: string;
  status: string;
  account_number: string;
  description: string;
  account_type: AccountTypeChoiceProps | "";
  cash_flow_type?: CashFlowTypeChoiceProps | "" | null;
  account_sub_type?: AccountSubTypeChoiceProps | "" | null;
  account_classification?: AccountClassificationChoiceProps | "" | null;
};

export interface GLAccountFormValues
  extends Omit<GeneralLedgerAccount, "id" | "organization"> {}

/** Types for Revenue Codes */
export type RevenueCode = {
  id: string;
  organization: string;
  code: string;
  description: string;
  expense_account?: string | null;
  revenue_account?: string | null;
};

export interface RevenueCodeFormValues
  extends Omit<RevenueCode, "id" | "organization"> {}
