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

export type TChoiceProps = {
  value: string;
  label: string;
};

export type ThemeOptions = "light" | "dark" | "slate-dark" | "system";

export interface IChoiceProps<T extends string | boolean> {
  value: T;
  label: string;
}

export interface BChoiceProps {
  value: boolean;
  label: string;
}

export type StatusChoiceProps = "A" | "I";

export type YesNoChoiceProps = "Y" | "N";

export type TDayOfWeekChoiceProps =
  | "MON"
  | "TUE"
  | "WED"
  | "THU"
  | "FRI"
  | "SAT"
  | "SUN";

type NestedKeys<T> = {
  [K in keyof T]: K extends string ? `${K}.${NestedKeys<T[K]>}` | K : never;
}[keyof T];

export type InputFieldNameProp<T> = keyof T | NestedKeys<T>;

/** Query Keys used in Monta by react-query
 *
 * @note: Only written to give autocomplete & type checking so people don't invalidate or use
 * query keys that don't exist. THANK ME LATER!
 */
export type QueryKeys =
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
  | "documentClassifications"
  | "depots"
  | "emailControl"
  | "equipment-manufacturer-table-data"
  | "equipmentManufacturers"
  | "equipment-type-table-data"
  | "equipmentTypes"
  | "feasibilityControl"
  | "fleet-code-table-data"
  | "fleetCodes"
  | "gl-account-table-data"
  | "glAccounts"
  | "hazardous-material-table-data"
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
  | "locations"
  | "locations-table-data"
  | "locationCategories"
  | "location-categories-table-data"
  | "tags"
  | "shipment-type-table-data"
  | "shipmentTypes"
  | "shipmentControl"
  | "service-type-table-data"
  | "serviceTypes"
  | "accountingControl"
  | "financeTransaction"
  | "reconciliationQueue"
  | "users"
  | "usStates"
  | "qualifier-code-table-data"
  | "qualifierCodes"
  | "worker-table-data";
