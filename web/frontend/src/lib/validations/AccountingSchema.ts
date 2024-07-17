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



import { StatusChoiceProps } from "@/types";
import {
  AccountingControlFormValues,
  DivisionCodeFormValues,
  GLAccountFormValues,
  RevenueCodeFormValues,
  TagFormValues,
} from "@/types/accounting";
import { ObjectSchema, array, boolean, number, object, string } from "yup";
import {
  AccountClassificationChoiceProps,
  AccountSubTypeChoiceProps,
  AccountTypeChoiceProps,
  AutomaticJournalEntryChoiceType,
  CashFlowTypeChoiceProps,
  ThresholdActionChoiceType,
} from "../choices";

export const revenueCodeSchema: ObjectSchema<RevenueCodeFormValues> =
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required"),
    code: string()
      .max(4, "Code cannot be longer than 4 characters.")
      .required("Code is required"),
    description: string().required("Description is required"),
    expenseAccountId: string().notRequired().nullable(),
    revenueAccountId: string().notRequired().nullable(),
  });

export const tagSchema: ObjectSchema<TagFormValues> = object().shape({
  id: string().required("Tag is required"),
});

export const glAccountSchema: ObjectSchema<GLAccountFormValues> =
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required"),
    accountNumber: string()
      .required("Account number is required")
      .max(7, "Account number cannot be longer than 7 characters.")
      .test(
        "account_number_format",
        "Account number must be in the format 0000-00",
        (value) => {
          if (!value) {
            return false;
          }
          const regex = /^\d{4}-\d{2}$/;
          return regex.test(value);
        },
      ),
    accountType: string<AccountTypeChoiceProps>().required(
      "Account type is required",
    ),
    cashFlowType: string<CashFlowTypeChoiceProps>().nullable().optional(),
    accountSubType: string<AccountSubTypeChoiceProps>().nullable().optional(),
    accountClassification: string<AccountClassificationChoiceProps>()
      .nullable()
      .optional(),
    notes: string().optional(),
    isReconciled: boolean(),
    isTaxRelevant: boolean(),
    interestRate: number().nullable().optional(),
    tagIds: array().of(string()),
    tags: array().of(tagSchema),
  });

export const divisionCodeSchema: ObjectSchema<DivisionCodeFormValues> =
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required"),
    code: string()
      .max(4, "Code cannot be longer than 4 characters")
      .required("Code is required"),
    description: string()
      .max(100, "Description cannot be longer than 100 characters")
      .required("Description is required"),
    apAccountId: string().notRequired(),
    cashAccountId: string().notRequired(),
    expenseAccountId: string().notRequired(),
  });

export const accountingControlSchema: ObjectSchema<AccountingControlFormValues> =
  object().shape({
    autoCreateJournalEntries: boolean().required(
      "Automatically Create Journal Entries must be yes or no",
    ),
    journalEntryCriteria: string<AutomaticJournalEntryChoiceType>().required(
      "Journal Entry Criteria is required",
    ),
    restrictManualJournalEntries: boolean().required(
      "Restrict Manual Journal Entries must be yes or no",
    ),
    requireJournalEntryApproval: boolean().required(
      "Require Journal Entry Approval must be yes or no",
    ),
    defaultRevAccountId: string().notRequired(),
    defaultExpAccountId: string().notRequired(),
    enableRecNotifications: boolean().required(
      "Enable Reconciliation Notifications must be yes or no",
    ),
    reconciliationNotificationRecipients: array().of(
      string().required("Reconciliation Notification Recipients is required"),
    ),
    recThreshold: number().required("Reconciliation Threshold is required"),
    recThresholdAction: string<ThresholdActionChoiceType>().required(
      "Reconciliation Threshold Action is required",
    ),
    haltOnPendingRec: boolean().required(
      "Halt on Pending Reconciliation must be yes or no",
    ),
    criticalProcesses: string(),
  });
