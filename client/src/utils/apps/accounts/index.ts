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

/** Types for Job Function Choices */
export type JobFunctionChoiceProps =
  | "MANAGER"
  | "MANAGEMENT_TRAINEE"
  | "SUPERVISOR"
  | "DISPATCHER"
  | "BILLING"
  | "FINANCE"
  | "SAFETY"
  | "SYS_ADMIN"
  | "TEST";

export const jobFunctionChoices = [
  { value: "MANAGER", label: "Manager" },
  { value: "MANAGEMENT_TRAINEE", label: "Management Trainee" },
  { value: "SUPERVISOR", label: "Supervisor" },
  { value: "DISPATCHER", label: "Dispatcher" },
  { value: "BILLING", label: "Billing" },
  { value: "FINANCE", label: "Finance" },
  { value: "SAFETY", label: "Safety" },
  { value: "SYS_ADMIN", label: "System Administrator" },
  { value: "TEST", label: "Test Job Function" },
];
