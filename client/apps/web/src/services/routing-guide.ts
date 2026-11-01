import {
  CreateRoutingGuideDocument,
  DeleteRoutingGuideDocument,
  UpdateRoutingGuideDocument,
  type RoutingGuideInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import {
  routingGuidePayloadSchema,
  type RoutingGuidePayload,
} from "@trenova/shared/types/routing-guide";

/** The guide and its whole waterfall, as the API takes them. */
export function toRoutingGuideInput(payload: RoutingGuidePayload): RoutingGuideInput {
  return {
    version: payload.version ?? null,
    name: payload.name,
    description: payload.description ?? "",
    status: payload.status,
    originLocationId: payload.originLocationId || null,
    destinationLocationId: payload.destinationLocationId || null,
    originCity: payload.originCity ?? "",
    originState: payload.originState ?? "",
    destinationCity: payload.destinationCity ?? "",
    destinationState: payload.destinationState ?? "",
    entries: payload.entries.map((entry) => ({
      id: entry.id || null,
      version: entry.version ?? null,
      carrierId: entry.carrierId,
      rank: entry.rank,
      rateMethod: entry.rateMethod,
      rate: entry.rate.toString(),
      useContractRate: entry.useContractRate,
      offerTtlSeconds: entry.offerTtlSeconds,
      channel: entry.channel,
    })),
  };
}

export class RoutingGuideService {
  public async create(payload: RoutingGuidePayload): Promise<RoutingGuidePayload> {
    const parsed = routingGuidePayloadSchema.parse(payload);
    const response = await requestGraphQL({
      document: CreateRoutingGuideDocument,
      operationName: "CreateRoutingGuide",
      variables: { input: toRoutingGuideInput(parsed) },
    });
    return { ...parsed, ...response.createRoutingGuide };
  }

  public async update(guideId: string, payload: RoutingGuidePayload): Promise<RoutingGuidePayload> {
    const parsed = routingGuidePayloadSchema.parse(payload);
    const response = await requestGraphQL({
      document: UpdateRoutingGuideDocument,
      operationName: "UpdateRoutingGuide",
      variables: { id: guideId, input: toRoutingGuideInput(parsed) },
    });
    return { ...parsed, ...response.updateRoutingGuide };
  }

  public async delete(guideId: string): Promise<void> {
    await requestGraphQL({
      document: DeleteRoutingGuideDocument,
      operationName: "DeleteRoutingGuide",
      variables: { id: guideId },
    });
  }
}
