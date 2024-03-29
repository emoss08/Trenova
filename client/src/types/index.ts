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

import { IconProp } from "@fortawesome/fontawesome-svg-core";

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

export type StatusChoiceProps = "A" | "I";

export type YesNoChoiceProps = "Y" | "N";

/** Query Keys used in Trenova by react-query
 *
 * @note: Only written to give autocomplete & type checking so people don't invalidate or use
 * query keys that don't exist. THANK ME LATER!
 */
export type QueryKey =
  | "accessorialCharges"
  | "accessorial-charges-table-data"
  | "billingControl"
  | "charge-type-table-data"
  | "chargeTypes"
  | "comment-types-table-data"
  | "commentTypes"
  | "commodity-table-data"
  | "commodities"
  | "customers-table-data"
  | "customers"
  | "dispatchControl"
  | "delay-code-table-data"
  | "delayCodes"
  | "division-code-table-data"
  | "divisionCodes"
  | "document-classification-table-data"
  | "documentClassifications"
  | "depots"
  | "emailControl"
  | "emailProfiles"
  | "equipment-manufacturer-table-data"
  | "equipmentManufacturers"
  | "equipment-type-table-data"
  | "equipmentTypes"
  | "email-profile-table-data"
  | "feasibilityControl"
  | "fleet-code-table-data"
  | "fleetCodes"
  | "gl-account-table-data"
  | "glAccounts"
  | "googleAPI"
  | "hazardous-material-table-data"
  | "hazardous-material-segregation-table-data"
  | "hazardousMaterials"
  | "invoiceControl"
  | "job-title-table-data"
  | "jobTitles"
  | "rate-table-data"
  | "rates"
  | "revenue-code-table-data"
  | "revenueCodes"
  | "reason-code-table-data"
  | "reasonCodes"
  | "routeControl"
  | "table-change-alert-data"
  | "tableNames"
  | "topicNames"
  | "trailers"
  | "trailer-table-data"
  | "tractors"
  | "tractor-table-data"
  | "locations"
  | "locationAutoComplete"
  | "locations-table-data"
  | "locationCategories"
  | "location-categories-table-data"
  | "tags"
  | "shipments"
  | "shipment-type-table-data"
  | "shipmentCountByStatus"
  | "shipmentTypes"
  | "shipmentControl"
  | "service-type-table-data"
  | "serviceTypes"
  | "accountingControl"
  | "financeTransaction"
  | "reconciliationQueue"
  | "users"
  | "usStates"
  | "userOrganization"
  | "userFavorites"
  | "qualifier-code-table-data"
  | "qualifierCodes"
  | "worker-table-data"
  | "workers"
  | "validateBOLNumber";

export type QueryKeys = [QueryKey];

export type QueryKeyWithParams<K extends QueryKey, Params extends unknown[]> = [
  K,
  ...Params,
];
