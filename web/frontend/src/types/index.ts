/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { type IconProp } from "@fortawesome/fontawesome-svg-core";

export type ValuesOf<T extends any[]> = T[number];

export type TChoiceProps = {
  value: string;
  label: string;
};

export type ThemeOptions = "light" | "dark" | "system";

export interface IChoiceProps<T extends string | boolean | number> {
  value: T;
  label: string;
  color?: string;
  description?: string;
  icon?: IconProp;
}

export type StatusChoiceProps = "Active" | "Inactive";

export type YesNoChoiceProps = "Y" | "N";

/** Query Keys used in Trenova by react-query
 *
 * @note: Only written to give autocomplete & type checking so people don't invalidate or use
 * query keys that don't exist. THANK ME LATER!
 */
export type QueryKey =
  | "authenticatedUser"
  | "accessorialCharges"
  | "billingControl"
  | "chargeTypes"
  | "commentTypes"
  | "commodities"
  | "customers"
  | "dispatchControl"
  | "delayCodes"
  | "divisionCodes"
  | "documentClassifications"
  | "depots"
  | "emailControl"
  | "emailProfiles"
  | "equipmentManufacturers"
  | "equipmentTypes"
  | "feasibilityControl"
  | "fleetCodes"
  | "glAccounts"
  | "googleAPI"
  | "hazardousMaterials"
  | "invoiceControl"
  | "jobTitles"
  | "rates"
  | "revenueCodes"
  | "reasonCodes"
  | "routeControl"
  | "tableNames"
  | "topicNames"
  | "trailers"
  | "tractors"
  | "locations"
  | "locationCategories"
  | "locationAutoComplete"
  | "tags"
  | "shipments"
  | "shipmentTypes"
  | "shipmentControl"
  | "serviceTypes"
  | "accountingControl"
  | "financeTransaction"
  | "reconciliationQueue"
  | "users"
  | "tableChangeAlerts"
  | "usStates"
  | "userRoles"
  | "hazardousMaterialsSegregations"
  | "userOrganization"
  | "userFavorites"
  | "qualifierCodes"
  | "workers"
  | "currentUser"
  | "reportColumns"
  | "organization"
  | "dailyShipmentCounts"
  | "shipmentCountByStatus"
  | "validateBOLNumber";

export type QueryKeys = [QueryKey];

export type QueryKeyWithParams<K extends QueryKey, Params extends unknown[]> = [
  K,
  ...Params,
];
