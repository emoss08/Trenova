import {
  ApplyRateIncreaseDocument,
  AppliedShipmentRateQuoteDocument,
  ApproveRateAgreementDocument,
  ApproveRateMatrixDocument,
  ArchiveRateAgreementDocument,
  ArchiveRateMatrixDocument,
  CommitRateImportDocument,
  CreateRateAgreementDocument,
  CreateRateMatrixDocument,
  CreateRateSimulationDocument,
  DiscardRateImportDocument,
  DuplicateRateAgreementDocument,
  PreviewRateIncreaseDocument,
  RateAgreementDetailDocument,
  RateAgreementLaneHistoryDocument,
  RateAgreementLanePageDocument,
  RateAgreementLaneSummaryDocument,
  RateAgreementSimulationsDocument,
  RateAgreementVersionsDocument,
  RateImportFailedRowsDocument,
  RateMatrixDetailDocument,
  RateMatrixDraftCellsDocument,
  RateMatrixReviewersDocument,
  RateMatrixVersionCellsDocument,
  RateSimulationDetailDocument,
  RateSimulationResultsDocument,
  RejectRateAgreementDocument,
  RejectRateMatrixDocument,
  ReplaceRateMatrixCellsDocument,
  RequestRateMatrixChangesDocument,
  RestoreRateMatrixVersionDocument,
  ResumeRateAgreementDocument,
  ShopShipmentCarriersDocument,
  SubmitRateAgreementDocument,
  SubmitRateMatrixDocument,
  SuspendRateAgreementDocument,
  UpdateRateAgreementDocument,
  UpdateRateMatrixDocument,
  type RateAgreementAccessorialInput,
  type RateAgreementFuelBindingInput,
  type RateAgreementInput,
  type RateAgreementRuleInput,
  type RateMatrixCellInput,
  type RateMatrixInput,
  type RateIncreaseInput,
} from "@trenova/graphql/generated/graphql";
import { z } from "zod";
import { api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  rateAgreementLanePageSchema,
  rateAgreementLaneSummarySchema,
  rateAgreementRuleSchema,
  rateAgreementSchema,
  rateAgreementVersionSchema,
  rateMatrixCellSchema,
  rateMatrixSchema,
  rateQuoteSchema,
  rateImportBatchSchema,
  rateImportRowSchema,
  rateIncreasePlanSchema,
  rateSimulationResultSchema,
  rateSimulationSchema,
  shopResultSchema,
  rateZoneSchema,
  type RateAgreement,
  type RateAgreementAccessorial,
  type RateAgreementFuelBinding,
  type RateAgreementLanePage,
  type RateAgreementLaneSummary,
  type RateAgreementRule,
  type RateAgreementVersion,
  type RateIncreasePlan,
  type RateMatrix,
  type RatePartyType,
  type RateMatrixCell,
  type RateQuote,
  type RateZone,
  type RateImportBatch,
  type RateImportRow,
  type RateSimulation,
  type RateSimulationResult,
  type ShopResult,
  type ShopStrategy,
} from "@trenova/shared/types/rate";
import {
  listFormulaReviewersResponseSchema,
  type FormulaReviewer,
} from "@trenova/shared/types/formula-template";

type RequestOptions = { signal?: AbortSignal };

/** What submitting a matrix draft sends: who should review it and a note for them. */
export type RateMatrixSubmitRequest = {
  comment?: string;
  reviewerId?: string;
  reviewerNote?: string;
};

/** The scope and size of a general rate increase, as the server takes it. */
export type RateIncreaseRequestPayload = {
  agreementIds?: string[];
  customerId?: string | null;
  carrierId?: string | null;
  partyType?: RatePartyType;
  adjustment: { percentChange: number };
  effectiveFrom: number;
};

/** One of the review steps an agreement moves through. */
export type RateAgreementReviewAction =
  | "submit"
  | "approve"
  | "reject"
  | "suspend"
  | "resume"
  | "archive";

/** A decimal as the API takes it: a string, or null when it is not set. */
function decimalInput(value: number | string | null | undefined): string | null {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  return String(value);
}

function idsInput(values: string[] | null | undefined): string[] | null {
  return values && values.length > 0 ? values : null;
}

function toRuleInput(rule: RateAgreementRule): RateAgreementRuleInput {
  return {
    id: rule.id || null,
    label: rule.label,
    status: rule.status,
    originScopeType: rule.originScopeType,
    originScopeValue: rule.originScopeValue,
    originCity: rule.originCity,
    destinationScopeType: rule.destinationScopeType,
    destinationScopeValue: rule.destinationScopeValue,
    destinationCity: rule.destinationCity,
    direction: rule.direction,
    originRadiusMeters: rule.originRadiusMeters ?? null,
    destinationRadiusMeters: rule.destinationRadiusMeters ?? null,
    originLatitude: rule.originLatitude ?? null,
    originLongitude: rule.originLongitude ?? null,
    destinationLatitude: rule.destinationLatitude ?? null,
    destinationLongitude: rule.destinationLongitude ?? null,
    serviceTypeIds: idsInput(rule.serviceTypeIds),
    shipmentTypeIds: idsInput(rule.shipmentTypeIds),
    tractorTypeIds: idsInput(rule.tractorTypeIds),
    trailerTypeIds: idsInput(rule.trailerTypeIds),
    commodityIds: idsInput(rule.commodityIds),
    freightClasses: idsInput(rule.freightClasses),
    serviceModels: idsInput(rule.serviceModels),
    equipmentClasses: idsInput(rule.equipmentClasses),
    minWeight: decimalInput(rule.minWeight),
    maxWeight: decimalInput(rule.maxWeight),
    minDistance: decimalInput(rule.minDistance),
    maxDistance: decimalInput(rule.maxDistance),
    minStops: rule.minStops ?? null,
    maxStops: rule.maxStops ?? null,
    daysOfWeek: rule.daysOfWeek,
    hazmatOnly: rule.hazmatOnly,
    tempControlOnly: rule.tempControlOnly,
    formulaTemplateId: rule.formulaTemplateId,
    inputs: rule.inputs,
    currency: rule.currency ?? null,
    priority: rule.priority,
    effectiveFrom: rule.effectiveFrom,
    effectiveTo: rule.effectiveTo ?? null,
  };
}

function toAccessorialInput(accessorial: RateAgreementAccessorial): RateAgreementAccessorialInput {
  return {
    accessorialChargeId: accessorial.accessorialChargeId,
    method: accessorial.method,
    rateUnit: accessorial.rateUnit ?? null,
    amount: decimalInput(accessorial.amount) ?? "",
    waived: accessorial.waived,
    autoApply: accessorial.autoApply,
    condition: accessorial.condition ?? null,
    freeUnits: accessorial.freeUnits ?? null,
    maxAmount: decimalInput(accessorial.maxAmount),
    serviceTypeIds: idsInput(accessorial.serviceTypeIds),
    shipmentTypeIds: idsInput(accessorial.shipmentTypeIds),
    effectiveFrom: accessorial.effectiveFrom ?? null,
    effectiveTo: accessorial.effectiveTo ?? null,
  };
}

function toFuelBindingInput(
  binding: RateAgreementFuelBinding | null | undefined,
): RateAgreementFuelBindingInput | null {
  if (!binding) {
    return null;
  }
  return {
    fuelSurchargeProgramId: binding.fuelSurchargeProgramId,
    waived: binding.waived,
    pegPriceOverride: decimalInput(binding.pegPriceOverride),
    incrementRateOverride: decimalInput(binding.incrementRateOverride),
    capAmount: decimalInput(binding.capAmount),
  };
}

/** The agreement as the editor saves it: header terms, every rule, the schedule and fuel. */
export function toRateAgreementInput(data: RateAgreement): RateAgreementInput {
  return {
    partyType: data.partyType,
    customerId: data.customerId ?? null,
    carrierId: data.carrierId ?? null,
    code: data.code,
    name: data.name,
    description: data.description,
    agreementType: data.agreementType,
    contractRef: data.contractRef,
    documentId: data.documentId ?? null,
    priority: data.priority,
    effectiveFrom: data.effectiveFrom,
    effectiveTo: data.effectiveTo ?? null,
    autoRenew: data.autoRenew,
    renewalNoticeDays: data.renewalNoticeDays,
    currency: data.currency,
    inputs: data.inputs,
    billToCustomerId: data.billToCustomerId ?? null,
    marginFloorPercent: decimalInput(data.marginFloorPercent),
    maxPayPercentOfSell: decimalInput(data.maxPayPercentOfSell),
    version: data.version ?? null,
    rules: data.rules.map(toRuleInput),
    accessorials: data.accessorials.map(toAccessorialInput),
    fuelBinding: toFuelBindingInput(data.fuelBinding),
  };
}

function toRateIncreaseInput(payload: RateIncreaseRequestPayload): RateIncreaseInput {
  return {
    agreementIds: idsInput(payload.agreementIds),
    customerId: payload.customerId ?? null,
    carrierId: payload.carrierId ?? null,
    partyType: payload.partyType ?? null,
    adjustment: { percentChange: decimalInput(payload.adjustment.percentChange) },
    effectiveFrom: payload.effectiveFrom,
  };
}

const reviewDocuments = {
  submit: { document: SubmitRateAgreementDocument, operationName: "SubmitRateAgreement" },
  approve: { document: ApproveRateAgreementDocument, operationName: "ApproveRateAgreement" },
  reject: { document: RejectRateAgreementDocument, operationName: "RejectRateAgreement" },
  suspend: { document: SuspendRateAgreementDocument, operationName: "SuspendRateAgreement" },
  resume: { document: ResumeRateAgreementDocument, operationName: "ResumeRateAgreement" },
  archive: { document: ArchiveRateAgreementDocument, operationName: "ArchiveRateAgreement" },
} as const;

/** The agreement after a write: the editor's values with the identity the server gave them. */
type SavedRecord = { id: string; version: number };

export class RateAgreementService {
  /** The agreement with every rule that has not been closed out, as the editor edits it. */
  public async getById(id: string, options?: RequestOptions): Promise<RateAgreement> {
    return this.read(id, true, options);
  }

  /**
   * The agreement with its accessorials and fuel binding but none of its lanes,
   * which a reader takes a page at a time with `listLanes`.
   */
  public async getWithoutLanes(id: string, options?: RequestOptions): Promise<RateAgreement> {
    return this.read(id, false, options);
  }

  private async read(
    id: string,
    withRules: boolean,
    options?: RequestOptions,
  ): Promise<RateAgreement> {
    const data = await requestGraphQL({
      document: RateAgreementDetailDocument,
      operationName: "RateAgreementDetail",
      variables: { id, withRules },
      signal: options?.signal,
    });

    return safeParse(rateAgreementSchema, data.rateAgreement, "Rate Agreement");
  }

  /** One page of the agreement's live lanes, most specific first, with the counts. */
  public async listLanes(
    id: string,
    page: { limit: number; offset: number },
    options?: RequestOptions,
  ): Promise<RateAgreementLanePage> {
    const data = await requestGraphQL({
      document: RateAgreementLanePageDocument,
      operationName: "RateAgreementLanePage",
      variables: { rateAgreementId: id, limit: page.limit, offset: page.offset },
      signal: options?.signal,
    });

    return safeParse(
      rateAgreementLanePageSchema,
      data.rateAgreementLanePage,
      "Rate Agreement Lanes",
    );
  }

  /** The agreement's live lanes grouped by what prices them, with each group's rate range. */
  public async laneSummary(
    id: string,
    options?: RequestOptions,
  ): Promise<RateAgreementLaneSummary> {
    const data = await requestGraphQL({
      document: RateAgreementLaneSummaryDocument,
      operationName: "RateAgreementLaneSummary",
      variables: { rateAgreementId: id },
      signal: options?.signal,
    });

    return safeParse(
      rateAgreementLaneSummarySchema,
      data.rateAgreementLaneSummary,
      "Rate Agreement Lane Summary",
    );
  }

  public async create(data: RateAgreement): Promise<RateAgreement> {
    const response = await requestGraphQL({
      document: CreateRateAgreementDocument,
      operationName: "CreateRateAgreement",
      variables: { input: toRateAgreementInput(data) },
    });

    return withSaved(data, response.createRateAgreement);
  }

  public async update(id: string, data: RateAgreement): Promise<RateAgreement> {
    const response = await requestGraphQL({
      document: UpdateRateAgreementDocument,
      operationName: "UpdateRateAgreement",
      variables: { id, input: toRateAgreementInput(data) },
    });

    return withSaved(data, response.updateRateAgreement);
  }

  /** The header terms as they stood at each renegotiation, newest first. */
  public async listVersions(id: string, options?: RequestOptions): Promise<RateAgreementVersion[]> {
    const data = await requestGraphQL({
      document: RateAgreementVersionsDocument,
      operationName: "RateAgreementVersions",
      variables: { rateAgreementId: id },
      signal: options?.signal,
    });

    return safeParse(
      z.array(rateAgreementVersionSchema),
      data.rateAgreementVersions,
      "Rate Agreement Versions",
    );
  }

  /**
   * One lane's full rate history — every rule that ever priced it, newest
   * first, closed-out windows included. This is the answer to "what did this
   * lane cost in March".
   */
  public async listRuleHistory(
    id: string,
    laneKey: string,
    options?: RequestOptions,
  ): Promise<RateAgreementRule[]> {
    const data = await requestGraphQL({
      document: RateAgreementLaneHistoryDocument,
      operationName: "RateAgreementLaneHistory",
      variables: { rateAgreementId: id, laneKey },
      signal: options?.signal,
    });

    return safeParse(
      z.array(rateAgreementRuleSchema),
      data.rateAgreementLaneHistory,
      "Rate Agreement Lane History",
    );
  }

  /**
   * Copies the whole contract — lanes, breaks, accessorials, fuel terms — as a
   * fresh draft. The renewal workflow starts here.
   */
  public async duplicate(id: string, payload: { code?: string; name?: string } = {}) {
    const response = await requestGraphQL({
      document: DuplicateRateAgreementDocument,
      operationName: "DuplicateRateAgreement",
      variables: { id, code: payload.code ?? null, name: payload.name ?? null },
    });

    return response.duplicateRateAgreement;
  }

  /** What a bulk rate change would do — before it does any of it. */
  public async previewRateIncrease(payload: RateIncreaseRequestPayload): Promise<RateIncreasePlan> {
    const data = await requestGraphQL({
      document: PreviewRateIncreaseDocument,
      operationName: "PreviewRateIncrease",
      variables: { input: toRateIncreaseInput(payload) },
    });

    return safeParse(rateIncreasePlanSchema, data.previewRateIncrease, "Rate Increase Plan");
  }

  /**
   * Applies the increase: every affected rule is closed out and succeeded at
   * the new rate from the effective date. The old rates stay in history.
   */
  public async applyRateIncrease(payload: RateIncreaseRequestPayload): Promise<RateIncreasePlan> {
    const data = await requestGraphQL({
      document: ApplyRateIncreaseDocument,
      operationName: "ApplyRateIncrease",
      variables: { input: toRateIncreaseInput(payload) },
    });

    return safeParse(rateIncreasePlanSchema, data.applyRateIncrease, "Rate Increase Plan");
  }

  /**
   * Moves an agreement to its next state. Every step takes the same arguments, so they
   * share one method rather than six that differ only in the mutation they send.
   */
  public async review(
    id: string,
    action: RateAgreementReviewAction,
    comment: string,
  ): Promise<void> {
    const { document, operationName } = reviewDocuments[action];
    await requestGraphQL({
      document: document as typeof SubmitRateAgreementDocument,
      operationName,
      variables: { id, comment },
    });
  }
}

function withSaved<T extends { id?: string; version?: number }>(data: T, saved: SavedRecord): T {
  return { ...data, id: saved.id, version: saved.version };
}

export class RateZoneService {
  public async getById(id: string): Promise<RateZone> {
    const response = await api.get<RateZone>(`/rate-zones/${id}/`);

    return safeParse(rateZoneSchema, response, "Rate Zone");
  }

  public async create(data: RateZone): Promise<RateZone> {
    const response = await api.post<RateZone>("/rate-zones/", data);

    return safeParse(rateZoneSchema, response, "Rate Zone");
  }

  public async update(id: string, data: RateZone): Promise<RateZone> {
    const response = await api.put<RateZone>(`/rate-zones/${id}/`, data);

    return safeParse(rateZoneSchema, response, "Rate Zone");
  }
}

export class RateMatrixService {
  public async getById(id: string, options?: RequestOptions): Promise<RateMatrix> {
    const data = await requestGraphQL({
      document: RateMatrixDetailDocument,
      operationName: "RateMatrixDetail",
      variables: { id },
      signal: options?.signal,
    });
    const matrix = data.rateMatrix;

    return safeParse(
      rateMatrixSchema,
      matrix ? { ...matrix, draftCellCount: matrix.cellCount } : matrix,
      "Rate Matrix",
    );
  }

  public async create(data: RateMatrix): Promise<RateMatrix> {
    const response = await requestGraphQL({
      document: CreateRateMatrixDocument,
      operationName: "CreateRateMatrix",
      variables: { input: toRateMatrixInput(data) },
    });

    return withSaved(data, response.createRateMatrix);
  }

  public async update(id: string, data: RateMatrix): Promise<RateMatrix> {
    const response = await requestGraphQL({
      document: UpdateRateMatrixDocument,
      operationName: "UpdateRateMatrix",
      variables: { id, input: toRateMatrixInput(data) },
    });

    return withSaved(data, response.updateRateMatrix);
  }

  /**
   * The whole grid of the draft, or of one published version.
   *
   * A sheet has to be read as a sheet — a hole in it is only visible against
   * the rows and columns around it — so the editor holds every cell rather
   * than a page of them. A page holds at most 100 cells and a twenty-by-twenty
   * zone tariff is four hundred, so this walks the pages.
   */
  public async listCells(
    id: string,
    versionNumber: number | null = null,
    options?: RequestOptions,
  ): Promise<RateMatrixCell[]> {
    const cells: RateMatrixCell[] = [];

    for (let offset = 0; ; offset += CELL_PAGE_SIZE) {
      const page = await this.cellPage(id, versionNumber, offset, options);
      for (const cell of page) {
        cells.push(await safeParse(rateMatrixCellSchema, cell, "Rate Matrix Cell"));
      }

      if (page.length < CELL_PAGE_SIZE) break;
    }

    return cells;
  }

  private async cellPage(
    id: string,
    versionNumber: number | null,
    offset: number,
    options?: RequestOptions,
  ): Promise<unknown[]> {
    if (versionNumber === null) {
      const data = await requestGraphQL({
        document: RateMatrixDraftCellsDocument,
        operationName: "RateMatrixDraftCells",
        variables: { id, offset, limit: CELL_PAGE_SIZE },
        signal: options?.signal,
      });
      return data.rateMatrix?.cells ?? [];
    }

    const data = await requestGraphQL({
      document: RateMatrixVersionCellsDocument,
      operationName: "RateMatrixVersionCells",
      variables: { id, versionNumber, offset, limit: CELL_PAGE_SIZE },
      signal: options?.signal,
    });
    return data.rateMatrixVersion?.cells ?? [];
  }

  /**
   * Swap the whole grid for this one.
   *
   * Merging would leave behind whatever the new sheet dropped, and a rate
   * nobody meant to keep is how a matrix ends up pricing a lane its author
   * believed they had removed.
   */
  public async replaceCells(id: string, cells: RateMatrixCell[]): Promise<void> {
    await requestGraphQL({
      document: ReplaceRateMatrixCellsDocument,
      operationName: "ReplaceRateMatrixCells",
      variables: { id, cells: cells.map(toRateMatrixCellInput) },
    });
  }

  /** Sends the draft to review; a reviewer publishing it makes it the next version. */
  public async submit(id: string, request: RateMatrixSubmitRequest) {
    const data = await requestGraphQL({
      document: SubmitRateMatrixDocument,
      operationName: "SubmitRateMatrix",
      variables: {
        id,
        input: {
          comment: request.comment ?? null,
          reviewerId: request.reviewerId || null,
          reviewerNote: request.reviewerNote ?? null,
        },
      },
    });
    return data.submitRateMatrix;
  }

  /** Publishes the submitted draft as the next version. */
  public async approve(id: string, comment: string) {
    const data = await requestGraphQL({
      document: ApproveRateMatrixDocument,
      operationName: "ApproveRateMatrix",
      variables: { id, comment },
    });
    return data.approveRateMatrix;
  }

  public async reject(id: string, comment: string) {
    const data = await requestGraphQL({
      document: RejectRateMatrixDocument,
      operationName: "RejectRateMatrix",
      variables: { id, comment },
    });
    return data.rejectRateMatrix;
  }

  public async requestChanges(id: string, comment: string) {
    const data = await requestGraphQL({
      document: RequestRateMatrixChangesDocument,
      operationName: "RequestRateMatrixChanges",
      variables: { id, comment },
    });
    return data.requestRateMatrixChanges;
  }

  /**
   * Takes the matrix out of new template versions. One that template versions pin needs
   * `confirmInUse`; they keep reading the version they pinned.
   */
  public async archive(id: string, confirmInUse: boolean) {
    const data = await requestGraphQL({
      document: ArchiveRateMatrixDocument,
      operationName: "ArchiveRateMatrix",
      variables: { id, confirmInUse },
    });
    return data.archiveRateMatrix;
  }

  /** Replaces the draft's axes and cells with a published version's. */
  public async restore(id: string, versionNumber: number) {
    const data = await requestGraphQL({
      document: RestoreRateMatrixVersionDocument,
      operationName: "RestoreRateMatrixVersion",
      variables: { id, versionNumber },
    });
    return data.restoreRateMatrixVersion;
  }

  public async listReviewers(id: string, options?: RequestOptions): Promise<FormulaReviewer[]> {
    const data = await requestGraphQL({
      document: RateMatrixReviewersDocument,
      operationName: "RateMatrixReviewers",
      variables: { rateMatrixId: id },
      signal: options?.signal,
    });
    return safeParse(
      listFormulaReviewersResponseSchema,
      data.rateMatrixReviewers,
      "Rate Matrix Reviewers",
    );
  }
}

const CELL_PAGE_SIZE = 100;

function toRateMatrixInput(data: RateMatrix): RateMatrixInput {
  return {
    code: data.code,
    name: data.name,
    description: data.description,
    valueType: data.valueType,
    version: data.version ?? null,
    dimensions: data.dimensions.map((dimension) => ({
      position: dimension.position,
      kind: dimension.kind,
      matchMode: dimension.matchMode,
      label: dimension.label,
      keyNormalization: dimension.keyNormalization,
      rangeOverflow: dimension.rangeOverflow,
    })),
  };
}

/**
 * One cell as the replace takes it, sending only what the cell sets: a large grid stays
 * well inside the request size limit when empty keys and bands are left out.
 */
export function toRateMatrixCellInput(cell: RateMatrixCell): RateMatrixCellInput {
  const input: RateMatrixCellInput = { value: decimalInput(cell.value) ?? "" };
  const keys = [
    "d0Key",
    "d1Key",
    "d2Key",
    "d3Key",
    "d0KeyTo",
    "d1KeyTo",
    "d2KeyTo",
    "d3KeyTo",
  ] as const;
  for (const key of keys) {
    if (cell[key]) input[key] = cell[key];
  }
  const bands = [
    "d0Min",
    "d0Max",
    "d1Min",
    "d1Max",
    "d2Min",
    "d2Max",
    "d3Min",
    "d3Max",
    "minCharge",
  ] as const;
  for (const band of bands) {
    const value = decimalInput(cell[band]);
    if (value !== null) input[band] = value;
  }
  if (!cell.deficitEligible) input.deficitEligible = false;
  return input;
}

export class RateQuoteService {
  /**
   * The quote a shipment is currently billed from, with its full trace, or null
   * for a shipment that has never been rated.
   */
  public async getAppliedForShipment(
    shipmentId: string,
    options?: RequestOptions,
  ): Promise<RateQuote | null> {
    const data = await requestGraphQL({
      document: AppliedShipmentRateQuoteDocument,
      operationName: "AppliedShipmentRateQuote",
      variables: { shipmentId },
      signal: options?.signal,
    });

    return safeParse(rateQuoteSchema.nullable(), data.appliedShipmentRateQuote, "Rate Quote");
  }

  /**
   * Prices this shipment against the carriers that could haul it and ranks
   * them. Shopping never saves a quote: a screen browsing options should not
   * leave one behind for every carrier somebody glanced at.
   */
  public async shop(
    shipmentId: string,
    options: {
      strategy?: ShopStrategy;
      carrierIds?: string[];
      asOf?: number;
      limit?: number;
    } = {},
    request?: RequestOptions,
  ): Promise<ShopResult> {
    const data = await requestGraphQL({
      document: ShopShipmentCarriersDocument,
      operationName: "ShopShipmentCarriers",
      variables: {
        shipmentId,
        input: {
          strategy: options.strategy ?? "LeastCost",
          carrierIds: idsInput(options.carrierIds),
          asOf: options.asOf ?? null,
          limit: options.limit ?? null,
        },
      },
      signal: request?.signal,
    });

    return safeParse(shopResultSchema, data.shopShipmentCarriers, "Shop Result");
  }
}

export class RateSimulationService {
  public async getById(id: string, options?: RequestOptions): Promise<RateSimulation> {
    const data = await requestGraphQL({
      document: RateSimulationDetailDocument,
      operationName: "RateSimulationDetail",
      variables: { id },
      signal: options?.signal,
    });

    return safeParse(rateSimulationSchema, data.rateSimulation, "Rate Simulation");
  }

  /** The simulations run against one agreement, newest first. */
  public async listForAgreement(
    rateAgreementId: string,
    options?: RequestOptions,
  ): Promise<RateSimulation[]> {
    const data = await requestGraphQL({
      document: RateAgreementSimulationsDocument,
      operationName: "RateAgreementSimulations",
      variables: { rateAgreementId },
      signal: options?.signal,
    });

    return safeParse(z.array(rateSimulationSchema), data.rateSimulations, "Rate Simulations");
  }

  /**
   * Records a simulation to be replayed.
   *
   * The run happens in the background: a year of shipments takes minutes, and a
   * request that waited for it would time out long before the answer exists.
   * Poll the returned simulation for its status.
   */
  public async create(payload: {
    rateAgreementId: string;
    name: string;
    description?: string;
    partyType?: RateSimulation["partyType"];
    sampleFrom: number;
    sampleTo: number;
    sampleLimit?: number;
  }): Promise<RateSimulation> {
    const data = await requestGraphQL({
      document: CreateRateSimulationDocument,
      operationName: "CreateRateSimulation",
      variables: {
        input: {
          rateAgreementId: payload.rateAgreementId,
          name: payload.name,
          description: payload.description ?? "",
          partyType: payload.partyType ?? "Customer",
          sampleFrom: payload.sampleFrom,
          sampleTo: payload.sampleTo,
          sampleLimit: payload.sampleLimit ?? 0,
        },
      },
    });

    return safeParse(rateSimulationSchema, data.createRateSimulation, "Rate Simulation");
  }

  /**
   * The per-shipment rows, largest increases first — the shipment that will
   * produce the phone call is what somebody opened this for.
   */
  public async listResults(
    id: string,
    options: { changedOnly?: boolean; limit?: number } = {},
    request?: RequestOptions,
  ): Promise<RateSimulationResult[]> {
    const data = await requestGraphQL({
      document: RateSimulationResultsDocument,
      operationName: "RateSimulationResults",
      variables: {
        rateSimulationId: id,
        changedOnly: options.changedOnly ?? false,
        first: options.limit ?? 50,
      },
      signal: request?.signal,
    });

    return safeParse(
      z.array(rateSimulationResultSchema),
      data.rateSimulationResults,
      "Rate Simulation Results",
    );
  }
}

export class RateImportService {
  /** The starter sheet a person fills in instead of guessing at column names. */
  public async template(): Promise<{ fileName: string; content: string }> {
    return api.get<{ fileName: string; content: string }>("/rate-imports/template/");
  }

  /**
   * Reads a sheet and stages what committing it would do.
   *
   * Nothing about the agreement changes here — what comes back is the dry run,
   * and applying it takes a second, deliberate call.
   */
  public async upload(payload: {
    rateAgreementId: string;
    effectiveFrom: number;
    file: File;
  }): Promise<RateImportBatch> {
    const form = new FormData();
    form.append("file", payload.file);
    form.append("rateAgreementId", payload.rateAgreementId);
    form.append("effectiveFrom", String(payload.effectiveFrom));

    const response = await api.upload<RateImportBatch>("/rate-imports/", form);

    return safeParse(rateImportBatchSchema, response, "Rate Import");
  }

  /** Applies a reviewed sheet to its agreement. */
  public async commit(id: string): Promise<RateImportBatch> {
    const data = await requestGraphQL({
      document: CommitRateImportDocument,
      operationName: "CommitRateImport",
      variables: { id },
    });

    return safeParse(rateImportBatchSchema, data.commitRateImport, "Rate Import");
  }

  /** Closes a sheet somebody read and said no to. */
  public async discard(id: string): Promise<RateImportBatch> {
    const data = await requestGraphQL({
      document: DiscardRateImportDocument,
      operationName: "DiscardRateImport",
      variables: { id },
    });

    return safeParse(rateImportBatchSchema, data.discardRateImport, "Rate Import");
  }

  /** The rows that would not read, which is what somebody opens an import to fix. */
  public async listFailedRows(id: string, options?: RequestOptions): Promise<RateImportRow[]> {
    const data = await requestGraphQL({
      document: RateImportFailedRowsDocument,
      operationName: "RateImportFailedRows",
      variables: { rateImportId: id },
      signal: options?.signal,
    });

    return safeParse(z.array(rateImportRowSchema), data.rateImportRows, "Rate Import Rows");
  }
}
