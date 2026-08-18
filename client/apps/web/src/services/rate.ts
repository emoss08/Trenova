import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  rateAgreementSchema,
  rateMatrixSchema,
  rateQuoteSchema,
  ratedShipmentSchema,
  rateZoneSchema,
  type RateAgreement,
  type RateMatrix,
  type RateQuote,
  type RatedShipment,
  type RateZone,
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
}
