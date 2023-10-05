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

import * as Yup from "yup";
import { ObjectSchema } from "yup";
import {
  DivisionCodeFormValues,
  GLAccountFormValues,
  RevenueCodeFormValues,
} from "@/types/accounting";
import { StatusChoiceProps } from "@/types";
import {
  AccountClassificationChoiceProps,
  AccountSubTypeChoiceProps,
  AccountTypeChoiceProps,
  CashFlowTypeChoiceProps,
} from "../choices";

export const revenueCodeSchema: ObjectSchema<RevenueCodeFormValues> =
  Yup.object().shape({
    code: Yup.string()
      .max(4, "Code cannot be longer than 4 characters.")
      .required("Code is required"),
    description: Yup.string().required("Description is required"),
    expenseAccount: Yup.string().notRequired(),
    revenueAccount: Yup.string().notRequired(),
  });

export const glAccountSchema: ObjectSchema<GLAccountFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    accountNumber: Yup.string()
      .required("Code is required")
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
    description: Yup.string()
      .max(100, "Description cannot be longer than 100 characters")
      .required("Description is required"),
    accountType: Yup.string<AccountTypeChoiceProps>().required(
      "Account type is required",
    ),
    cashFlowType: Yup.string<CashFlowTypeChoiceProps>().notRequired(),
    accountSubType: Yup.string<AccountSubTypeChoiceProps>().notRequired(),
    accountClassification:
      Yup.string<AccountClassificationChoiceProps>().notRequired(),
    parentAccount: Yup.string().notRequired(),
    isReconciled: Yup.boolean().required("Reconciled is required"),
    notes: Yup.string().notRequired(),
    owner: Yup.string().notRequired(),
    isTaxRelevant: Yup.boolean().required("Tax relevant is required"),
    attachment: Yup.mixed().notRequired(),
    interestRate: Yup.number().notRequired(),
    tags: Yup.array().notRequired(),
  });

export const divisionCodeSchema: ObjectSchema<DivisionCodeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(4, "Code cannot be longer than 4 characters")
      .required("Code is required"),
    description: Yup.string()
      .max(100, "Description cannot be longer than 100 characters")
      .required("Description is required"),
    apAccount: Yup.string().notRequired(),
    cashAccount: Yup.string().notRequired(),
    expenseAccount: Yup.string().notRequired(),
  });
