import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  rateAgreementSchema,
  rateMatrixCellSchema,
  rateMatrixSchema,
  rateQuoteSchema,
  ratedShipmentSchema,
  rateSimulationSchema,
  shopResultSchema,
  rateZoneSchema,
  type RateAgreement,
  type RateMatrix,
  type RateMatrixCell,
  type RateQuote,
  type RatedShipment,
  type RateZone,
  type RateSimulation,
  type RateSimulationResult,
  type ShopResult,
  type ShopStrategy,
} from "@trenova/shared/types/rate";

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
