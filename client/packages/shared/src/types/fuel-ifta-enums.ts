import { z } from "zod";

// Wire values mirror the GraphQL enums IFTAFuelType, FuelCardProvider,
// FuelCardStatus, FuelQuantityUnit, FuelPurchaseSource,
// FuelPurchaseImportStatus, IFTAMileageSource and IFTAReturnStatus. Codegen
// emits those as bare union types, so the runtime lists live here and the
// `toXInput` mappers make any drift a typecheck error.

export const iftaFuelTypeSchema = z.enum([
  "Diesel",
  "Gasoline",
  "Gasohol",
  "Propane",
  "CNG",
  "LNG",
  "Ethanol",
  "Methanol",
  "E85",
  "M85",
  "A55",
  "Biodiesel",
  "Electricity",
  "Hydrogen",
  "DEF",
  "Reefer",
  "Other",
]);
export type IftaFuelType = z.infer<typeof iftaFuelTypeSchema>;

export const IFTA_FUEL_TYPE_LABELS: Record<IftaFuelType, string> = {
  Diesel: "Diesel",
  Gasoline: "Gasoline",
  Gasohol: "Gasohol",
  Propane: "Propane",
  CNG: "Compressed natural gas (CNG)",
  LNG: "Liquefied natural gas (LNG)",
  Ethanol: "Ethanol",
  Methanol: "Methanol",
  E85: "E85 (85% ethanol)",
  M85: "M85 (85% methanol)",
  A55: "A55 (55% methanol)",
  Biodiesel: "Biodiesel",
  Electricity: "Electricity",
  Hydrogen: "Hydrogen",
  DEF: "Diesel exhaust fluid (DEF)",
  Reefer: "Reefer fuel",
  Other: "Other",
};

export const fuelCardProviderSchema = z.enum(["Comdata", "EFS", "WEX", "Other"]);
export type FuelCardProvider = z.infer<typeof fuelCardProviderSchema>;

export const FUEL_CARD_PROVIDER_LABELS: Record<FuelCardProvider, string> = {
  Comdata: "Comdata",
  EFS: "EFS",
  WEX: "WEX",
  Other: "Other",
};

export const fuelCardStatusSchema = z.enum(["Active", "Suspended", "Cancelled"]);
export type FuelCardStatus = z.infer<typeof fuelCardStatusSchema>;

export const FUEL_CARD_STATUS_LABELS: Record<FuelCardStatus, string> = {
  Active: "Active",
  Suspended: "Suspended",
  Cancelled: "Cancelled",
};

export const fuelQuantityUnitSchema = z.enum(["Gallon", "Litre"]);
export type FuelQuantityUnit = z.infer<typeof fuelQuantityUnitSchema>;

export const FUEL_QUANTITY_UNIT_LABELS: Record<FuelQuantityUnit, string> = {
  Gallon: "Gallons",
  Litre: "Litres",
};

export const fuelPurchaseSourceSchema = z.enum(["Manual", "CardImport"]);
export type FuelPurchaseSource = z.infer<typeof fuelPurchaseSourceSchema>;

export const FUEL_PURCHASE_SOURCE_LABELS: Record<FuelPurchaseSource, string> = {
  Manual: "Manual",
  CardImport: "Card import",
};

export const fuelPurchaseImportStatusSchema = z.enum([
  "Pending",
  "Parsed",
  "Committed",
  "Failed",
  "Discarded",
]);
export type FuelPurchaseImportStatus = z.infer<typeof fuelPurchaseImportStatusSchema>;

export const FUEL_PURCHASE_IMPORT_STATUS_LABELS: Record<FuelPurchaseImportStatus, string> = {
  Pending: "Pending",
  Parsed: "Parsed",
  Committed: "Committed",
  Failed: "Failed",
  Discarded: "Discarded",
};

export const iftaMileageSourceSchema = z.enum(["Manual", "RouteCalculation", "Telematics"]);
export type IftaMileageSource = z.infer<typeof iftaMileageSourceSchema>;

export const IFTA_MILEAGE_SOURCE_LABELS: Record<IftaMileageSource, string> = {
  Manual: "Manual",
  RouteCalculation: "Route calculation",
  Telematics: "Telematics",
};

export const iftaReturnStatusSchema = z.enum(["Draft", "Finalized", "Filed"]);
export type IftaReturnStatus = z.infer<typeof iftaReturnStatusSchema>;

export const IFTA_RETURN_STATUS_LABELS: Record<IftaReturnStatus, string> = {
  Draft: "Draft",
  Finalized: "Finalized",
  Filed: "Filed",
};

export const iftaQuarterSchema = z.enum(["1", "2", "3", "4"]);
export type IftaQuarter = z.infer<typeof iftaQuarterSchema>;

export const IFTA_QUARTER_LABELS: Record<IftaQuarter, string> = {
  "1": "Q1 (Jan–Mar)",
  "2": "Q2 (Apr–Jun)",
  "3": "Q3 (Jul–Sep)",
  "4": "Q4 (Oct–Dec)",
};
