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

import { IconDefinition } from "@fortawesome/pro-duotone-svg-icons";

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
  icon?: IconDefinition;
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
