import {
  createRecurringShipmentGraphQL,
  generateRecurringShipmentGraphQL,
  listRecurringShipmentRunsGraphQL,
  matchRecurringShipmentsGraphQL,
  updateRecurringShipmentGraphQL,
  updateRecurringShipmentStatusGraphQL,
} from "@/lib/graphql/recurring-shipment";
import type { RecurringShipmentInput } from "@trenova/graphql/generated/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  generateRecurringShipmentResultSchema,
  matchRecurringShipmentsResponseSchema,
  recurringShipmentRunSchema,
  recurringShipmentSchema,
  type RecurringShipment,
  type RecurringShipmentStatus,
} from "@/types/recurring-shipment";

export type MatchRecurringShipmentsInput = {
  customerId: string;
  originLocationId: string;
  destinationLocationId: string;
};

export function toRecurringShipmentInput(data: RecurringShipment): RecurringShipmentInput {
  return {
    sourceShipmentId: data.sourceShipmentId,
    name: data.name,
    description: data.description || null,
    status: data.status,
    cronExpression: data.cronExpression,
    timezone: data.timezone,
    startDate: data.startDate ?? null,
    endDate: data.endDate ?? null,
    maxOccurrences: data.maxOccurrences ?? null,
    leadTimeDays: data.leadTimeDays,
    skipWeekends: data.skipWeekends,
    exceptionPolicy: data.exceptionPolicy,
    blackoutDates: data.blackoutDates ?? null,
    autoGenerate: data.autoGenerate,
    version: data.version ?? null,
  };
}

export class RecurringShipmentService {
  async create(data: RecurringShipment) {
    const response = await createRecurringShipmentGraphQL(toRecurringShipmentInput(data));
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async update(id: string, data: RecurringShipment) {
    const response = await updateRecurringShipmentGraphQL(id, toRecurringShipmentInput(data));
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async updateStatus(id: string, status: RecurringShipmentStatus, version: number) {
    const response = await updateRecurringShipmentStatusGraphQL(id, status, version);
    return safeParse(recurringShipmentSchema, response, "RecurringShipment");
  }

  async generate(id: string, occurrenceAt?: number | null) {
    const response = await generateRecurringShipmentGraphQL(id, occurrenceAt);
    return safeParse(
      generateRecurringShipmentResultSchema,
      response,
      "GenerateRecurringShipmentResult",
    );
  }

  async match(input: MatchRecurringShipmentsInput, { signal }: { signal?: AbortSignal } = {}) {
    const response = await matchRecurringShipmentsGraphQL(input, { signal });
    return safeParse(
      matchRecurringShipmentsResponseSchema,
      response,
      "MatchRecurringShipmentsResponse",
    );
  }

  async listRuns(id: string, limit: number, { signal }: { signal?: AbortSignal } = {}) {
    const response = await listRecurringShipmentRunsGraphQL(id, limit, { signal });
    return {
      results: await safeParse(
        recurringShipmentRunSchema.array(),
        response,
        "RecurringShipmentRuns",
      ),
    };
  }
}
