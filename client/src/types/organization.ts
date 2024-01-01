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

export type Organization = {
  id: string;
  name: string;
  scacCode: string;
  dotNumber: number;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  state?: string;
  zipCode?: string;
  phoneNumber?: string;
  website?: string;
  orgType: string;
  timezone: string;
  language: string;
  currency: string;
  dateFormat: string;
  timeFormat: string;
  logo?: string;
  darkLogo?: string;
  tokenExpirationDays: number;
};

export type EmailProfile = {
  id: string;
  name: string;
  organization: string;
  email: string;
  protocol: string;
  host: string;
  port: number;
  username: string;
  password: string;
};

export type Department = {
  id: string;
  name: string;
  organization: string;
  description: string;
  depot: string;
};

/** Types for EmailControl */
export type EmailControl = {
  id: string;
  organization: string;
  billingEmailProfile?: string | null;
  rateExpirationEmailProfile?: string | null;
};

export type EmailControlFormValues = {
  billingEmailProfile?: string | null;
  rateExpirationEmailProfile?: string | null;
};

export type Depot = BaseModel & {
  id: string;
  name: string;
  description?: string;
};

export type FeatureFlag = {
  name: string;
  code: string;
  description: string;
  enabled: boolean;
  beta: boolean;
  preview: string;
  paidOnly: boolean;
};

/** Base Monta Interface
 *
 * @note This interface is used for all Monta models that have the following fields:
 * - organization
 * - created
 * - modified
 *
 * Please do not put businessUnit in this interface. Add it directly to the interface that
 * extends this interface.
 * */
export type BaseModel = {
  organization: string;
  created: string;
  modified: string;
};
