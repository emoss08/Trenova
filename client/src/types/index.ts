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

export type StatusChoiceProps = "A" | "I";

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
  | "dailyShipmentCounts"
  | "validateBOLNumber";

export type QueryKeys = [QueryKey];

export type QueryKeyWithParams<K extends QueryKey, Params extends unknown[]> = [
  K,
  ...Params,
];
