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

import { IChoiceProps, TDayOfWeekChoiceProps } from "@/types";

/** Type for Account Type Choices */
export type AccountTypeChoiceProps =
  | "ASSET"
  | "LIABILITY"
  | "EQUITY"
  | "REVENUE"
  | "EXPENSE";

export const accountTypeChoices: ReadonlyArray<
  IChoiceProps<AccountTypeChoiceProps>
> = [
  { value: "ASSET", label: "Asset" },
  { value: "LIABILITY", label: "Liability" },
  { value: "EQUITY", label: "Equity" },
  { value: "REVENUE", label: "Revenue" },
  { value: "EXPENSE", label: "Expense" },
];

/** Type for Cash Flow Type Choices */
export type CashFlowTypeChoiceProps = "OPERATING" | "INVESTING" | "FINANCING";

export const cashFlowTypeChoices: ReadonlyArray<
  IChoiceProps<CashFlowTypeChoiceProps>
> = [
  { value: "OPERATING", label: "Operating" },
  { value: "INVESTING", label: "Investing" },
  { value: "FINANCING", label: "Financing" },
];

/** Type for Account Sub Type Choices */
export type AccountSubTypeChoiceProps =
  | "CURRENT_ASSET"
  | "FIXED_ASSET"
  | "OTHER_ASSET"
  | "CURRENT_LIABILITY"
  | "LONG_TERM_LIABILITY"
  | "EQUITY"
  | "REVENUE"
  | "COST_OF_GOODS_SOLD"
  | "EXPENSE"
  | "OTHER_INCOME"
  | "OTHER_EXPENSE";

export const accountSubTypeChoices: ReadonlyArray<
  IChoiceProps<AccountSubTypeChoiceProps>
> = [
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

/** Type for Account Classification Choices */
export type AccountClassificationChoiceProps =
  | "BANK"
  | "CASH"
  | "ACCOUNTS_RECEIVABLE"
  | "ACCOUNTS_PAYABLE"
  | "INVENTORY"
  | "OTHER_CURRENT_ASSET"
  | "FIXED_ASSET";

export const accountClassificationChoices = [
  { value: "BANK", label: "Bank" },
  { value: "CASH", label: "Cash" },
  { value: "ACCOUNTS_RECEIVABLE", label: "Accounts Receivable" },
  { value: "ACCOUNTS_PAYABLE", label: "Accounts Payable" },
  { value: "INVENTORY", label: "Inventory" },
  { value: "OTHER_CURRENT_ASSET", label: "Other Current Asset" },
  { value: "FIXED_ASSET", label: "Fixed Asset" },
] satisfies ReadonlyArray<IChoiceProps<AccountClassificationChoiceProps>>;

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
] satisfies ReadonlyArray<IChoiceProps<JobFunctionChoiceProps>>;

export const DayOfWeekChoices = [
  { value: "MON", label: "Monday" },
  { value: "TUE", label: "Tuesday" },
  { value: "WED", label: "Wednesday" },
  { value: "THU", label: "Thursday" },
  { value: "FRI", label: "Friday" },
  { value: "SAT", label: "Saturday" },
  { value: "SUN", label: "Sunday" },
] satisfies ReadonlyArray<IChoiceProps<TDayOfWeekChoiceProps>>;

type ServiceIncidentControlChoiceProps =
  | "Never"
  | "Pickup"
  | "Delivery"
  | "Pickup and Delivery"
  | "All except shipper";

export const ServiceIncidentControlChoices = [
  { value: "Never", label: "Never" },
  { value: "Pickup", label: "Pickup" },
  { value: "Delivery", label: "Delivery" },
  { value: "Pickup and Delivery", label: "Pickup and Delivery" },
  { value: "All except shipper", label: "All except shipper" },
] satisfies ReadonlyArray<IChoiceProps<ServiceIncidentControlChoiceProps>>;

/** Type for Date Format Choices */
export type DateFormatChoiceProps =
  | "%m/%d/%Y"
  | "%d/%m/%Y"
  | "%Y/%d/%m"
  | "%Y/%m/%d"
  | "";

export const DateFormatChoices = [
  { value: "%m/%d/%Y", label: "MM/DD/YYYY" },
  { value: "%d/%m/%Y", label: "DD/MM/YYYY" },
  { value: "%Y/%d/%m", label: "YYYY/DD/MM" },
  { value: "%Y/%m/%d", label: "YYYY/MM/DD" },
] satisfies ReadonlyArray<IChoiceProps<DateFormatChoiceProps>>;

/** Type for Route Avoidance Choices */
type RouteAvoidanceChoiceProps = "tolls" | "highways" | "ferries";

export const routeAvoidanceChoices = [
  { value: "tolls", label: "Tolls" },
  { value: "highways", label: "Highways" },
  { value: "ferries", label: "Ferries" },
] satisfies ReadonlyArray<IChoiceProps<RouteAvoidanceChoiceProps>>;

/** Type for Route Model Choices */
export type RouteModelChoiceProps = "best_guess" | "optimistic" | "pessimistic";

export const routeModelChoices = [
  { value: "best_guess", label: "Best Guess" },
  { value: "optimistic", label: "Optimistic" },
  { value: "pessimistic", label: "Pessimistic" },
] satisfies ReadonlyArray<IChoiceProps<RouteModelChoiceProps>>;

/** Type for Route Distance Unit Choices */
type RouteDistanceUnitProps = "metric" | "imperial";

export const routeDistanceUnitChoices = [
  { value: "metric", label: "Metric" },
  { value: "imperial", label: "Imperial" },
] satisfies ReadonlyArray<IChoiceProps<RouteDistanceUnitProps>>;

/** Type for Distance Method Choices */
export type DistanceMethodChoiceProps = "Google" | "Monta";

export const distanceMethodChoices = [
  { value: "Google", label: "Google" },
  { value: "Monta", label: "Monta" },
] satisfies ReadonlyArray<IChoiceProps<DistanceMethodChoiceProps>>;

/** Type for Feasibility Operator Choices */
type FeasibilityOperatorChoiceProps = "eq" | "ne" | "gt" | "gte" | "lt" | "lte";

export const feasibilityOperatorChoices = [
  { value: "eq", label: "Equals" },
  { value: "ne", label: "Not Equals" },
  { value: "gt", label: "Greater Than" },
  { value: "gte", label: "Greater Than or Equal To" },
  { value: "lt", label: "Less Than" },
  { value: "lte", label: "Less Than or Equal To" },
] satisfies ReadonlyArray<IChoiceProps<FeasibilityOperatorChoiceProps>>;

/** Type for Equipment Class Choices */
export type EquipmentClassChoiceProps =
  | "UNDEFINED"
  | "CAR"
  | "VAN"
  | "PICKUP"
  | "WALK_IN"
  | "STRAIGHT"
  | "TRACTOR"
  | "TRAILER";

export const EquipmentClassChoices = [
  { value: "UNDEFINED", label: "Undefined" },
  { value: "CAR", label: "Car" },
  { value: "VAN", label: "Van" },
  { value: "PICKUP", label: "Pickup" },
  { value: "WALK_IN", label: "Walk In" },
  { value: "STRAIGHT", label: "Straight" },
  { value: "TRACTOR", label: "Tractor" },
  { value: "TRAILER", label: "Trailer" },
] satisfies ReadonlyArray<IChoiceProps<EquipmentClassChoiceProps>>;

/** Type for Unit of Measure Choices */
export type UnitOfMeasureChoiceProps =
  | "PALLET"
  | "TOTE"
  | "DRUM"
  | "CYLINDER"
  | "CASE"
  | "AMPULE"
  | "BAG"
  | "BOTTLE"
  | "PAIL"
  | "PIECES"
  | "ISO_TANK";

export const UnitOfMeasureChoices = [
  { value: "PALLET", label: "Pallet" },
  { value: "TOTE", label: "Tote" },
  { value: "DRUM", label: "Drum" },
  { value: "CYLINDER", label: "Cylinder" },
  { value: "CASE", label: "Case" },
  { value: "AMPULE", label: "Ampule" },
  { value: "BAG", label: "Bag" },
  { value: "BOTTLE", label: "Bottle" },
  { value: "PAIL", label: "Pail" },
  { value: "PIECES", label: "Pieces" },
  { value: "ISO_TANK", label: "ISO Tank" },
] satisfies ReadonlyArray<IChoiceProps<UnitOfMeasureChoiceProps>>;

/** Type for Hazardous Class Choices */
export type PackingGroupChoiceProps = "I" | "II" | "III";

export const PackingGroupChoices = [
  { value: "I", label: "I" },
  { value: "II", label: "II" },
  { value: "III", label: "III" },
] satisfies ReadonlyArray<IChoiceProps<PackingGroupChoiceProps>>;

/** Type for Hazardous Class Choices */
export type HazardousClassChoiceProps =
  | "1.1"
  | "1.2"
  | "1.3"
  | "1.4"
  | "1.5"
  | "1.6"
  | "2.1"
  | "2.2"
  | "2.3"
  | "3"
  | "4.1"
  | "4.2"
  | "4.3"
  | "5.1"
  | "5.2"
  | "6.1"
  | "6.2"
  | "7"
  | "8"
  | "9";

export const HazardousClassChoices = [
  { value: "1.1", label: "Division 1.1: Mass Explosive Hazard" },
  { value: "1.2", label: "Division 1.2: Projection Hazard" },
  {
    value: "1.3",
    label: "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
  },
  { value: "1.4", label: "Division 1.4: Minor Explosion Hazard" },
  {
    value: "1.5",
    label: "Division 1.5: Very Insensitive With Mass Explosion Hazard",
  },
  {
    value: "1.6",
    label: "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
  },
  { value: "2.1", label: "Division 2.1: Flammable Gases" },
  { value: "2.2", label: "Division 2.2: Non-Flammable Gases" },
  { value: "2.3", label: "Division 2.3: Poisonous Gases" },
  { value: "3", label: "Division 3: Flammable Liquids" },
  { value: "4.1", label: "Division 4.1: Flammable Solids" },
  { value: "4.2", label: "Division 4.2: Spontaneously Combustible Solids" },
  { value: "4.3", label: "Division 4.3: Dangerous When Wet" },
  { value: "5.1", label: "Division 5.1: Oxidizing Substances" },
  { value: "5.2", label: "Division 5.2: Organic Peroxides" },
  { value: "6.1", label: "Division 6.1: Toxic Substances" },
  { value: "6.2", label: "Division 6.2: Infectious Substances" },
  { value: "7", label: "Division 7: Radioactive Material" },
  { value: "8", label: "Division 8: Corrosive Substances" },
  {
    value: "9",
    label: "Division 9: Miscellaneous Hazardous Substances and Articles",
  },
] satisfies ReadonlyArray<IChoiceProps<HazardousClassChoiceProps>>;
