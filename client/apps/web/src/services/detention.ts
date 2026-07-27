import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  deskEntrySchema,
  backtestResultSchema,
  customerDetentionStatSchema,
  facilityDetentionStatSchema,
  previewResultSchema,
  waiverLeakageStatSchema,
  detentionOccurrenceSchema,
  disputePacketSchema,
  occurrenceDetailSchema,
  type DeskEntry,
  type DetentionOccurrence,
  type DisputePacket,
  type BacktestResult,
  type CustomerDetentionStat,
  type DetentionPolicy,
  type DetentionStatsQuery,
  type FacilityDetentionStat,
  type WaiverLeakageStat,
  type OccurrenceDetail,
  type PreviewResult,
  type PreviewScenario,
  type WaiverReason,
} from "@trenova/shared/types/detention";
import { z } from "zod";

const deskEntryListSchema = z.array(deskEntrySchema);

export class DetentionService {
  public async desk(): Promise<DeskEntry[]> {
    const response = await api.get<DeskEntry[]>("/detention/desk/");
    return safeParse(deskEntryListSchema, response, "Detention Desk");
  }

  public async getOccurrence(id: string): Promise<OccurrenceDetail> {
    const response = await api.get<OccurrenceDetail>(
      `/detention/occurrences/${id}/`,
    );
    return safeParse(occurrenceDetailSchema, response, "Detention Occurrence");
  }

  public async byShipment(shipmentId: string): Promise<DetentionOccurrence[]> {
    const response = await api.get<{ results: DetentionOccurrence[] }>(
      `/detention/occurrences/?shipmentId=${shipmentId}&limit=100`,
    );

    return safeParse(
      z.array(detentionOccurrenceSchema),
      response?.results ?? [],
      "Detention Occurrences",
    );
  }

  public async disputePacket(id: string): Promise<DisputePacket> {
    const response = await api.get<DisputePacket>(
      `/detention/occurrences/${id}/dispute-packet/`,
    );
    return safeParse(disputePacketSchema, response, "Dispute Packet");
  }

  public async waive(
    id: string,
    payload: { reason: WaiverReason; note: string },
  ): Promise<DetentionOccurrence> {
    const response = await api.post<DetentionOccurrence>(
      `/detention/occurrences/${id}/waive/`,
      payload,
    );
    return safeParse(detentionOccurrenceSchema, response, "Detention Occurrence");
  }

  public async approve(id: string): Promise<DetentionOccurrence> {
    const response = await api.post<DetentionOccurrence>(
      `/detention/occurrences/${id}/approve/`,
      {},
    );
    return safeParse(detentionOccurrenceSchema, response, "Detention Occurrence");
  }

  public async dispute(
    id: string,
    payload: { note: string },
  ): Promise<DetentionOccurrence> {
    const response = await api.post<DetentionOccurrence>(
      `/detention/occurrences/${id}/dispute/`,
      payload,
    );
    return safeParse(detentionOccurrenceSchema, response, "Detention Occurrence");
  }
}

function statsQuery(params: DetentionStatsQuery = {}): string {
  const search = new URLSearchParams();
  if (params.from) search.set("from", String(params.from));
  if (params.to) search.set("to", String(params.to));
  if (params.limit) search.set("limit", String(params.limit));
  const query = search.toString();
  return query ? `?${query}` : "";
}

export class DetentionAnalyticsService {
  public async backtest(
    policy: DetentionPolicy,
    params: {
      from: number;
      to: number;
      assumeNoticeCompliance?: boolean;
      driverPayRate?: string;
    },
  ): Promise<BacktestResult> {
    const response = await api.post<BacktestResult>("/detention/backtest/", {
      policy,
      ...params,
    });

    return safeParse(backtestResultSchema, response, "Detention Backtest");
  }

  public async facilities(
    params: DetentionStatsQuery = {},
  ): Promise<FacilityDetentionStat[]> {
    const response = await api.get<FacilityDetentionStat[]>(
      `/detention/facilities/${statsQuery(params)}`,
    );

    return safeParse(
      z.array(facilityDetentionStatSchema),
      response ?? [],
      "Facility Detention Stats",
    );
  }

  public async customers(
    params: DetentionStatsQuery = {},
  ): Promise<CustomerDetentionStat[]> {
    const response = await api.get<CustomerDetentionStat[]>(
      `/detention/customers/${statsQuery(params)}`,
    );

    return safeParse(
      z.array(customerDetentionStatSchema),
      response ?? [],
      "Customer Detention Stats",
    );
  }

  public async waivers(params: DetentionStatsQuery = {}): Promise<WaiverLeakageStat[]> {
    const response = await api.get<WaiverLeakageStat[]>(
      `/detention/waivers/${statsQuery(params)}`,
    );

    return safeParse(
      z.array(waiverLeakageStatSchema),
      response ?? [],
      "Detention Waiver Leakage",
    );
  }
}

export class DetentionPolicyService {
  /**
   * Runs the server's production calculator against a hypothetical stop so the
   * builder shows what the configured terms would actually charge. The math is
   * never re-derived on the client.
   */
  public async preview(
    policy: DetentionPolicy,
    scenario: PreviewScenario,
  ): Promise<PreviewResult> {
    const response = await api.post<PreviewResult>("/detention-policies/preview/", {
      policy,
      scenario,
    });

    return safeParse(previewResultSchema, response, "Detention Preview");
  }
}
