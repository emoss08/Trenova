import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  rateAgreementSchema,
  rateMatrixCellSchema,
  rateMatrixSchema,
  rateQuoteSchema,
  ratedShipmentSchema,
  rateImportBatchSchema,
  rateIncreasePlanSchema,
  rateSimulationSchema,
  shopResultSchema,
  rateZoneSchema,
  type RateAgreement,
  type RateAgreementRule,
  type RateAgreementVersion,
  type RateIncreasePlan,
  type RateMatrix,
  type RatePartyType,
  type RateMatrixCell,
  type RateQuote,
  type RatedShipment,
  type RateZone,
  type RateImportBatch,
  type RateImportRow,
  type RateSimulation,
  type RateSimulationResult,
  type ShopResult,
  type ShopStrategy,
} from "@trenova/shared/types/rate";

/** The scope and size of a general rate increase, as the server takes it. */
export type RateIncreaseRequestPayload = {
  agreementIds?: string[];
  customerId?: string | null;
  carrierId?: string | null;
  partyType?: RatePartyType;
  adjustment: {
    percentChange?: number | null;
    flatChange?: number | null;
  };
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

export class RateAgreementService {
  public async getById(id: string): Promise<RateAgreement> {
    const response = await api.get<RateAgreement>(`/rate-agreements/${id}/?includeChildren=true`);

    return safeParse(rateAgreementSchema, response, "Rate Agreement");
  }

  public async create(data: RateAgreement): Promise<RateAgreement> {
    const response = await api.post<RateAgreement>("/rate-agreements/", data);

    return safeParse(rateAgreementSchema, response, "Rate Agreement");
  }

  public async update(id: string, data: RateAgreement): Promise<RateAgreement> {
    const response = await api.put<RateAgreement>(`/rate-agreements/${id}/`, data);

    return safeParse(rateAgreementSchema, response, "Rate Agreement");
  }

  /** The header terms as they stood at each renegotiation, newest first. */
  public async listVersions(id: string): Promise<RateAgreementVersion[]> {
    const response = await api.get<{ results?: RateAgreementVersion[] }>(
      `/rate-agreements/${id}/versions/`,
    );

    return response?.results ?? [];
  }

  /**
   * One lane's full rate history — every rule that ever priced it, newest
   * first, closed-out windows included. This is the answer to "what did this
   * lane cost in March".
   */
  public async listRuleHistory(id: string, laneKey: string): Promise<RateAgreementRule[]> {
    const query = new URLSearchParams({ laneKey, includeSuperseded: "true" });
    const response = await api.get<RateAgreementRule[]>(
      `/rate-agreements/${id}/rules/?${query.toString()}`,
    );

    return response ?? [];
  }

  /**
   * Copies the whole contract — lanes, breaks, accessorials, fuel terms — as a
   * fresh draft. The renewal workflow starts here.
   */
  public async duplicate(
    id: string,
    payload: { code?: string; name?: string } = {},
  ): Promise<RateAgreement> {
    const response = await api.post<RateAgreement>(`/rate-agreements/${id}/duplicate/`, payload);

    return safeParse(rateAgreementSchema, response, "Rate Agreement");
  }

  /** What a bulk rate change would do — before it does any of it. */
  public async previewRateIncrease(payload: RateIncreaseRequestPayload): Promise<RateIncreasePlan> {
    const response = await api.post<RateIncreasePlan>(
      "/rate-agreements/rate-increase/preview/",
      payload,
    );

    return safeParse(rateIncreasePlanSchema, response, "Rate Increase Plan");
  }

  /**
   * Applies the increase: every affected rule is closed out and succeeded at
   * the new rate from the effective date. The old rates stay in history.
   */
  public async applyRateIncrease(payload: RateIncreaseRequestPayload): Promise<RateIncreasePlan> {
    const response = await api.post<RateIncreasePlan>(
      "/rate-agreements/rate-increase/apply/",
      payload,
    );

    return safeParse(rateIncreasePlanSchema, response, "Rate Increase Plan");
  }

  /**
   * Moves an agreement to its next state.
   *
   * Every step takes the same request and returns the same thing, so they share
   * one method rather than six that differ only in the path.
   */
  public async review(
    id: string,
    action: RateAgreementReviewAction,
    comment: string,
  ): Promise<RateAgreement> {
    const response = await api.post<RateAgreement>(`/rate-agreements/${id}/${action}/`, {
      comment,
    });

    return safeParse(rateAgreementSchema, response, "Rate Agreement");
  }
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
  public async getById(id: string): Promise<RateMatrix> {
    const response = await api.get<RateMatrix>(`/rate-matrices/${id}/`);

    return safeParse(rateMatrixSchema, response, "Rate Matrix");
  }

  public async create(data: RateMatrix): Promise<RateMatrix> {
    const response = await api.post<RateMatrix>("/rate-matrices/", data);

    return safeParse(rateMatrixSchema, response, "Rate Matrix");
  }

  public async update(id: string, data: RateMatrix): Promise<RateMatrix> {
    const response = await api.put<RateMatrix>(`/rate-matrices/${id}/`, data);

    return safeParse(rateMatrixSchema, response, "Rate Matrix");
  }

  /**
   * The whole grid.
   *
   * A sheet has to be read as a sheet — a hole in it is only visible against
   * the rows and columns around it — so the editor holds every cell rather
   * than a page of them. The API caps a page at 100 and a twenty-by-twenty
   * zone tariff is four hundred cells, so this walks the pages.
   */
  public async listCells(id: string): Promise<RateMatrixCell[]> {
    const pageSize = 100;
    const cells: RateMatrixCell[] = [];

    for (let offset = 0; ; offset += pageSize) {
      const page = await api.get<{ results?: RateMatrixCell[]; count?: number }>(
        `/rate-matrices/${id}/cells/?limit=${pageSize}&offset=${offset}`,
      );

      const results = page?.results ?? [];
      for (const cell of results) {
        cells.push(await safeParse(rateMatrixCellSchema, cell, "Rate Matrix Cell"));
      }

      if (results.length < pageSize) break;
    }

    return cells;
  }

  /**
   * Swap the whole grid for this one.
   *
   * Merging would leave behind whatever the new sheet dropped, and a rate
   * nobody meant to keep is how a matrix ends up pricing a lane its author
   * believed they had removed.
   */
  public async replaceCells(id: string, cells: RateMatrixCell[]): Promise<void> {
    await api.put(`/rate-matrices/${id}/cells/`, { cells });
  }
}

export class RateQuoteService {
  /** The quote a shipment is currently billed from, with its full trace. */
  public async getAppliedForShipment(shipmentId: string): Promise<RateQuote> {
    const response = await api.get<RateQuote>(`/rate-quotes/shipment/${shipmentId}/applied/`);

    return safeParse(rateQuoteSchema, response, "Rate Quote");
  }

  /** A shipment's rating history, newest first. */
  public async listForShipment(shipmentId: string): Promise<RateQuote[]> {
    return api.get<RateQuote[]>(`/rate-quotes/shipment/${shipmentId}/`);
  }

  /**
   * Re-resolves a shipment against the contracts effective on a date, without
   * changing what it is billed. Omit `asOf` to reproduce its current rate.
   */
  public async explain(shipmentId: string, asOf?: number): Promise<RatedShipment> {
    const response = await api.post<RatedShipment>(`/rate-quotes/shipment/${shipmentId}/explain/`, {
      asOf: asOf ?? 0,
    });

    return safeParse(ratedShipmentSchema, response, "Rated Shipment");
  }

  /**
   * Prices this shipment against the carriers that could haul it and ranks
   * them.
   *
   * Persisting is off by default: a screen browsing options should not leave a
   * quote behind for every carrier somebody glanced at. The caller turns it on
   * when the shopping is the record of a decision.
   */
  public async shop(
    shipmentId: string,
    options: {
      strategy?: ShopStrategy;
      carrierIds?: string[];
      asOf?: number;
      limit?: number;
      persist?: boolean;
    } = {},
  ): Promise<ShopResult> {
    const response = await api.post<ShopResult>(`/rate-quotes/shipment/${shipmentId}/shop/`, {
      strategy: options.strategy ?? "LeastCost",
      carrierIds: options.carrierIds ?? [],
      asOf: options.asOf ?? 0,
      limit: options.limit ?? 0,
      persist: options.persist ?? false,
    });

    return safeParse(shopResultSchema, response, "Shop Result");
  }
}

export class RateSimulationService {
  public async getById(id: string): Promise<RateSimulation> {
    const response = await api.get<RateSimulation>(`/rate-simulations/${id}/`);

    return safeParse(rateSimulationSchema, response, "Rate Simulation");
  }

  /** The simulations run against one agreement, newest first. */
  public async listForAgreement(rateAgreementId: string): Promise<RateSimulation[]> {
    const response = await api.get<{ results?: RateSimulation[] }>(
      `/rate-simulations/?rateAgreementId=${encodeURIComponent(rateAgreementId)}&limit=20`,
    );

    return response?.results ?? [];
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
    const response = await api.post<RateSimulation>("/rate-simulations/", {
      ...payload,
      description: payload.description ?? "",
      partyType: payload.partyType ?? "Customer",
      sampleLimit: payload.sampleLimit ?? 0,
    });

    return safeParse(rateSimulationSchema, response, "Rate Simulation");
  }

  /**
   * The per-shipment rows, largest increases first — the shipment that will
   * produce the phone call is what somebody opened this for.
   */
  public async listResults(
    id: string,
    options: { changedOnly?: boolean; limit?: number } = {},
  ): Promise<RateSimulationResult[]> {
    const query = new URLSearchParams({
      changedOnly: String(options.changedOnly ?? false),
      limit: String(options.limit ?? 50),
    });

    const response = await api.get<{ results?: RateSimulationResult[] }>(
      `/rate-simulations/${id}/results/?${query.toString()}`,
    );

    return response?.results ?? [];
  }
}

export class RateImportService {
  public async getById(id: string): Promise<RateImportBatch> {
    const response = await api.get<RateImportBatch>(`/rate-imports/${id}/`);

    return safeParse(rateImportBatchSchema, response, "Rate Import");
  }

  /** The starter sheet a person fills in instead of guessing at column names. */
  public async template(): Promise<{ fileName: string; content: string }> {
    return api.get<{ fileName: string; content: string }>("/rate-imports/template/");
  }

  /** The sheets uploaded against one agreement, newest first. */
  public async listForAgreement(rateAgreementId: string): Promise<RateImportBatch[]> {
    const response = await api.get<{ results?: RateImportBatch[] }>(
      `/rate-imports/?rateAgreementId=${encodeURIComponent(rateAgreementId)}&limit=20`,
    );

    return response?.results ?? [];
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
    const response = await api.post<RateImportBatch>(`/rate-imports/${id}/commit/`, {});

    return safeParse(rateImportBatchSchema, response, "Rate Import");
  }

  /** Closes a sheet somebody read and said no to. */
  public async discard(id: string): Promise<RateImportBatch> {
    const response = await api.post<RateImportBatch>(`/rate-imports/${id}/discard/`, {});

    return safeParse(rateImportBatchSchema, response, "Rate Import");
  }

  /** The rows that would not read, which is what somebody opens an import to fix. */
  public async listFailedRows(id: string): Promise<RateImportRow[]> {
    const response = await api.get<{ results?: RateImportRow[] }>(
      `/rate-imports/${id}/rows/?failedOnly=true&limit=100`,
    );

    return response?.results ?? [];
  }
}
