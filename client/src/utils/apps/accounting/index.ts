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

export const accountTypeChoices = [
  { value: "ASSET", label: "Asset" },
  { value: "LIABILITY", label: "Liability" },
  { value: "EQUITY", label: "Equity" },
  { value: "REVENUE", label: "Revenue" },
  { value: "EXPENSE", label: "Expense" },
];

export const cashFlowTypeChoices = [
  { value: "OPERATING", label: "Operating" },
  { value: "INVESTING", label: "Investing" },
  { value: "FINANCING", label: "Financing" },
];

export const accountSubTypeChoices = [
  { value: "CURRENT_ASSET", label: "Current Asset" },
  { value: "FIXED_ASSET", label: "Fixed Asset" },
  { value: "OTHER_ASSET", label: "Other Asset" },
  { value: "CURRENT_LIABILITY", label: "Current Liability" },
  { value: "LONG_TERM_LIABILITY", label: "Long Term Liability" },
  { value: "EQUITY", label: "Equity" },
  { value: "REVENUE", label: "Revenue" },
  { value: "COST_OF_GOODS_SOLD", label: "Cost of Goods Sold" },
  { value: "EXPENSE", label: "Expense" },
  { value: "OTHER_INCOME", label: "Other Income" },
  { value: "OTHER_EXPENSE", label: "Other Expense" },
];

export const accountClassificationChoices = [
  { value: "BANK", label: "Bank" },
  { value: "CASH", label: "Cash" },
  { value: "ACCOUNTS_RECEIVABLE", label: "Accounts Receivable" },
  { value: "ACCOUNTS_PAYABLE", label: "Accounts Payable" },
  { value: "INVENTORY", label: "Inventory" },
  { value: "OTHER_CURRENT_ASSET", label: "Other Current Asset" },
  { value: "FIXED_ASSET", label: "Fixed Asset" },
];
