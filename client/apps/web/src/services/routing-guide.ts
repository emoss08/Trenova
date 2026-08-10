import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  routingGuidePayloadSchema,
  routingGuideSchema,
  type RoutingGuide,
  type RoutingGuidePayload,
} from "@trenova/shared/types/routing-guide";

type RoutingGuideEntryRequestBody = {
  id?: string;
  version?: number;
  carrierId: string;
  rank: number;
  rateMethod: RoutingGuidePayload["entries"][number]["rateMethod"];
  rate: string;
  offerTtlSeconds: number;
  channel: RoutingGuidePayload["entries"][number]["channel"];
};

type RoutingGuideRequestBody = {
  version?: number;
  name: string;
  description: string;
  status: RoutingGuidePayload["status"];
  originLocationId?: string;
  destinationLocationId?: string;
  originCity: string;
  originState: string;
  destinationCity: string;
  destinationState: string;
  entries: RoutingGuideEntryRequestBody[];
};

export function toRoutingGuideRequestBody(payload: RoutingGuidePayload): RoutingGuideRequestBody {
  return {
    version: payload.version,
    name: payload.name,
    description: payload.description ?? "",
    status: payload.status,
    originLocationId: payload.originLocationId ?? undefined,
    destinationLocationId: payload.destinationLocationId ?? undefined,
    originCity: payload.originCity ?? "",
    originState: payload.originState ?? "",
    destinationCity: payload.destinationCity ?? "",
    destinationState: payload.destinationState ?? "",
    entries: payload.entries.map((entry) => ({
      id: entry.id || undefined,
      version: entry.version,
      carrierId: entry.carrierId,
      rank: entry.rank,
      rateMethod: entry.rateMethod,
      rate: entry.rate.toString(),
      offerTtlSeconds: entry.offerTtlSeconds,
      channel: entry.channel,
    })),
  };
}

export class RoutingGuideService {
  public async get(guideId: string): Promise<RoutingGuide> {
    const response = await api.get<RoutingGuide>(`/routing-guides/${guideId}/`);
    return safeParse(routingGuideSchema, response, "RoutingGuide");
  }

  public async create(payload: RoutingGuidePayload): Promise<RoutingGuide> {
    const response = await api.post<RoutingGuide>(
      "/routing-guides/",
      toRoutingGuideRequestBody(routingGuidePayloadSchema.parse(payload)),
    );
    return safeParse(routingGuideSchema, response, "RoutingGuide");
  }

  public async update(guideId: string, payload: RoutingGuidePayload): Promise<RoutingGuide> {
    const response = await api.put<RoutingGuide>(
      `/routing-guides/${guideId}/`,
      toRoutingGuideRequestBody(routingGuidePayloadSchema.parse(payload)),
    );
    return safeParse(routingGuideSchema, response, "RoutingGuide");
  }

  public async delete(guideId: string): Promise<void> {
    await api.delete(`/routing-guides/${guideId}/`);
  }
}
