import {
  AddFuelIndexPriceDocument,
  CreateFuelIndexDocument,
  CreateFuelSurchargeProgramDocument,
  DeleteFuelIndexDocument,
  DeleteFuelIndexPriceDocument,
  DeleteFuelSurchargeProgramDocument,
  EiaSeriesOptionsDocument,
  FuelDashboardDocument,
  FuelIndexPriceHistoryDocument,
  FuelProgramCurrentRatesDocument,
  FuelSurchargeProgramDetailDocument,
  GenerateFuelSurchargeTableDocument,
  UpdateFuelIndexDocument,
  UpdateFuelIndexPriceDocument,
  UpdateFuelSurchargeProgramDocument,
  type EiaSeriesOptionsQuery,
  type FuelDashboardQuery,
  type FuelIndexInput,
  type FuelIndexPriceHistoryQuery,
  type FuelProgramCurrentRatesQuery,
  type FuelSurchargeProgramDetailQuery,
  type FuelSurchargeProgramInput,
  type GenerateFuelSurchargeTableQuery,
  type GenerateFuelTableInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import type { FuelIndex, FuelSurchargeProgramFormValues } from "@/types/fuel-surcharge";

export type FuelDashboardEntry = FuelDashboardQuery["fuelDashboard"][number];
export type FuelPriceHistoryEntry =
  FuelIndexPriceHistoryQuery["fuelIndexPriceHistory"][number];
export type FuelProgramCurrentRate =
  FuelProgramCurrentRatesQuery["fuelProgramCurrentRates"][number];
export type FuelSurchargeProgramDetail = NonNullable<
  FuelSurchargeProgramDetailQuery["fuelSurchargeProgram"]
>;
export type GeneratedFuelRow =
  GenerateFuelSurchargeTableQuery["generateFuelSurchargeTable"][number];
export type EIASeriesOption = EiaSeriesOptionsQuery["eiaSeriesOptions"][number];

const toDecimalString = (value: number | null | undefined): string | undefined =>
  value === null || value === undefined ? undefined : String(value);

function toFuelIndexInput(values: FuelIndex): FuelIndexInput {
  return {
    name: values.name,
    code: values.code,
    description: values.description || undefined,
    source: values.source,
    fuelType: values.fuelType,
    region: values.region || undefined,
    eiaSeriesId: values.source === "EIA" ? values.eiaSeriesId || undefined : undefined,
    currency: values.currency || undefined,
    isActive: values.isActive,
  };
}

function toProgramInput(values: FuelSurchargeProgramFormValues): FuelSurchargeProgramInput {
  const isTableMethod =
    values.method === "TablePerMile" ||
    values.method === "TablePercent" ||
    values.method === "TableFlat";

  return {
    name: values.name,
    code: values.code,
    description: values.description || undefined,
    status: values.status,
    fuelIndexId: values.fuelIndexId,
    accessorialChargeId: values.accessorialChargeId,
    method: values.method,
    pegPrice: toDecimalString(values.pegPrice),
    increment: toDecimalString(values.increment),
    incrementRate: toDecimalString(values.incrementRate),
    milesPerGallon: toDecimalString(values.milesPerGallon),
    percentBasis: values.percentBasis,
    stepRounding: values.stepRounding,
    rateRounding: values.rateRounding,
    ratePrecision: values.ratePrecision,
    minAmount: toDecimalString(values.minAmount),
    maxAmount: toDecimalString(values.maxAmount),
    dateBasis: values.dateBasis,
    priceEffectiveDay: values.priceEffectiveDay,
    missingPriceFallback: values.missingPriceFallback,
    effectiveStartDate: values.effectiveStartDate ?? undefined,
    effectiveEndDate: values.effectiveEndDate ?? undefined,
    shipmentTypeIds: values.shipmentTypeIds?.length ? values.shipmentTypeIds : undefined,
    serviceTypeIds: values.serviceTypeIds?.length ? values.serviceTypeIds : undefined,
    tractorTypeIds: values.tractorTypeIds?.length ? values.tractorTypeIds : undefined,
    trailerTypeIds: values.trailerTypeIds?.length ? values.trailerTypeIds : undefined,
    tableRows: isTableMethod
      ? values.tableRows.map((row, index) => ({
          priceMin: toDecimalString(row.priceMin),
          priceMax: toDecimalString(row.priceMax),
          value: String(row.value),
          sortOrder: index,
        }))
      : undefined,
  };
}

export async function fetchFuelDashboard() {
  const data = await requestGraphQL({
    document: FuelDashboardDocument,
    operationName: "FuelDashboard",
    variables: {},
  });
  return data.fuelDashboard;
}

export async function fetchFuelPriceHistory(
  indexId: string,
  options?: { from?: string; to?: string; limit?: number },
) {
  const data = await requestGraphQL({
    document: FuelIndexPriceHistoryDocument,
    operationName: "FuelIndexPriceHistory",
    variables: {
      indexId,
      from: options?.from,
      to: options?.to,
      limit: options?.limit,
    },
  });
  return data.fuelIndexPriceHistory;
}

export async function fetchFuelProgramCurrentRates() {
  const data = await requestGraphQL({
    document: FuelProgramCurrentRatesDocument,
    operationName: "FuelProgramCurrentRates",
    variables: {},
  });
  return data.fuelProgramCurrentRates;
}

export async function fetchFuelSurchargeProgramDetail(id: string) {
  const data = await requestGraphQL({
    document: FuelSurchargeProgramDetailDocument,
    operationName: "FuelSurchargeProgramDetail",
    variables: { id },
  });
  return data.fuelSurchargeProgram;
}

export async function generateFuelSurchargeTable(input: GenerateFuelTableInput) {
  const data = await requestGraphQL({
    document: GenerateFuelSurchargeTableDocument,
    operationName: "GenerateFuelSurchargeTable",
    variables: { input },
  });
  return data.generateFuelSurchargeTable;
}

export async function fetchEIASeriesOptions() {
  const data = await requestGraphQL({
    document: EiaSeriesOptionsDocument,
    operationName: "EIASeriesOptions",
    variables: {},
  });
  return data.eiaSeriesOptions;
}

export async function createFuelIndex(values: FuelIndex) {
  const data = await requestGraphQL({
    document: CreateFuelIndexDocument,
    operationName: "CreateFuelIndex",
    variables: { input: toFuelIndexInput(values) },
  });
  return data.createFuelIndex;
}

export async function updateFuelIndex(id: string, values: FuelIndex) {
  const data = await requestGraphQL({
    document: UpdateFuelIndexDocument,
    operationName: "UpdateFuelIndex",
    variables: { id, input: toFuelIndexInput(values) },
  });
  return data.updateFuelIndex;
}

export async function deleteFuelIndex(id: string) {
  const data = await requestGraphQL({
    document: DeleteFuelIndexDocument,
    operationName: "DeleteFuelIndex",
    variables: { id },
  });
  return data.deleteFuelIndex;
}

export async function addFuelIndexPrice(input: {
  fuelIndexId: string;
  priceDate: string;
  price: string;
}) {
  const data = await requestGraphQL({
    document: AddFuelIndexPriceDocument,
    operationName: "AddFuelIndexPrice",
    variables: { input },
  });
  return data.addFuelIndexPrice;
}

export async function updateFuelIndexPrice(input: {
  id: string;
  priceDate: string;
  price: string;
}) {
  const data = await requestGraphQL({
    document: UpdateFuelIndexPriceDocument,
    operationName: "UpdateFuelIndexPrice",
    variables: { input },
  });
  return data.updateFuelIndexPrice;
}

export async function deleteFuelIndexPrice(id: string) {
  const data = await requestGraphQL({
    document: DeleteFuelIndexPriceDocument,
    operationName: "DeleteFuelIndexPrice",
    variables: { id },
  });
  return data.deleteFuelIndexPrice;
}

export async function createFuelSurchargeProgram(values: FuelSurchargeProgramFormValues) {
  const data = await requestGraphQL({
    document: CreateFuelSurchargeProgramDocument,
    operationName: "CreateFuelSurchargeProgram",
    variables: { input: toProgramInput(values) },
  });
  return data.createFuelSurchargeProgram;
}

export async function updateFuelSurchargeProgram(
  id: string,
  values: FuelSurchargeProgramFormValues,
) {
  const data = await requestGraphQL({
    document: UpdateFuelSurchargeProgramDocument,
    operationName: "UpdateFuelSurchargeProgram",
    variables: { id, input: toProgramInput(values) },
  });
  return data.updateFuelSurchargeProgram;
}

export async function deleteFuelSurchargeProgram(id: string) {
  const data = await requestGraphQL({
    document: DeleteFuelSurchargeProgramDocument,
    operationName: "DeleteFuelSurchargeProgram",
    variables: { id },
  });
  return data.deleteFuelSurchargeProgram;
}
