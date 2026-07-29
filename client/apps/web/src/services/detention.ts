import {
  ApproveDetentionOccurrenceDocument,
  CreateDetentionPolicyDocument,
  DeleteDetentionPolicyDocument,
  DetentionBacktestDocument,
  DetentionCustomerStatsDocument,
  DetentionDeskDocument,
  DetentionDisputePacketDocument,
  DetentionFacilityStatsDocument,
  DetentionOccurrenceDetailDocument,
  DetentionPolicyDocument,
  DetentionPolicyPreviewDocument,
  DetentionWaiverStatsDocument,
  DisputeDetentionOccurrenceDocument,
  SendDetentionNoticeDocument,
  ShipmentDetentionDocument,
  UpdateDetentionPolicyDocument,
  WaiveDetentionOccurrenceDocument,
  type ApproveDetentionOccurrenceMutation,
  type CreateDetentionPolicyMutation,
  type DeleteDetentionPolicyMutation,
  type DetentionBacktestMutation,
  type DetentionCustomerStatsQuery,
  type DetentionDeskQuery,
  type DetentionDisputePacketQuery,
  type DetentionFacilityStatsQuery,
  type DetentionOccurrenceDetailQuery,
  type DetentionPolicyInput,
  type DetentionPolicyPreviewQuery,
  type DetentionPolicyQuery,
  type DetentionWaiverStatsQuery,
  type DisputeDetentionOccurrenceMutation,
  type SendDetentionNoticeMutation,
  type ShipmentDetentionQuery,
  type StopScheduleType,
  type StopType,
  type UpdateDetentionPolicyMutation,
  type WaiveDetentionOccurrenceMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  backtestResultSchema,
  customerDetentionStatSchema,
  deskEntrySchema,
  detentionOccurrenceSchema,
  detentionPolicySchema,
  disputePacketSchema,
  facilityDetentionStatSchema,
  occurrenceDetailSchema,
  previewResultSchema,
  waiverLeakageStatSchema,
  type BacktestResult,
  type CustomerDetentionStat,
  type DeskEntry,
  type DetentionOccurrence,
  type DetentionPolicy,
  type DisputePacket,
  type FacilityDetentionStat,
  type OccurrenceDetail,
  type PreviewResult,
  type PreviewScenario,
  type WaiverLeakageStat,
  type WaiverReason,
} from "@trenova/shared/types/detention";
import { z } from "zod";

const deskEntryListSchema = z.array(deskEntrySchema);

function decimalInput(value: number | null | undefined): string | null {
  return value === null || value === undefined ? null : String(value);
}

export function toDetentionPolicyInput(data: DetentionPolicy): DetentionPolicyInput {
  return {
    name: data.name,
    code: data.code,
    description: data.description ?? "",
    status: data.status,
    isOrgDefault: data.isOrgDefault,
    priority: data.priority,
    customerId: data.customerId || null,
    locationId: data.locationId || null,
    shipmentTypeIds: data.shipmentTypeIds?.length ? data.shipmentTypeIds : null,
    serviceTypeIds: data.serviceTypeIds?.length ? data.serviceTypeIds : null,
    commodityIds: data.commodityIds?.length ? data.commodityIds : null,
    stopTypes: data.stopTypes?.length ? (data.stopTypes as StopType[]) : null,
    appointmentStopsOnly: data.appointmentStopsOnly,
    effectiveStartDate: data.effectiveStartDate ?? null,
    effectiveEndDate: data.effectiveEndDate ?? null,
    clockStartBasis: data.clockStartBasis,
    lateArrivalRule: data.lateArrivalRule,
    lateArrivalGraceMinutes: data.lateArrivalGraceMinutes,
    billingFreeMinutes: data.billingFreeMinutes,
    pickupFreeMinutes: data.pickupFreeMinutes ?? null,
    deliveryFreeMinutes: data.deliveryFreeMinutes ?? null,
    payFreeMinutes: data.payFreeMinutes ?? null,
    minimumBillableMinutes: data.minimumBillableMinutes,
    billingIncrementMinutes: data.billingIncrementMinutes,
    roundingMode: data.roundingMode,
    rateSource: data.rateSource,
    accessorialChargeId: data.accessorialChargeId,
    tiers: data.tiers?.length
      ? data.tiers.map((tier, index) => ({
          fromMinute: tier.fromMinute,
          toMinute: tier.toMinute ?? null,
          rate: String(tier.rate),
          rateUnit: tier.rateUnit,
          label: tier.label ?? "",
          sortOrder: tier.sortOrder ?? index,
        }))
      : null,
    maxBillableMinutesPerStop: data.maxBillableMinutesPerStop ?? null,
    maxChargePerStop: decimalInput(data.maxChargePerStop),
    maxChargePerDay: decimalInput(data.maxChargePerDay),
    maxChargePerShipment: decimalInput(data.maxChargePerShipment),
    dayBoundaryMode: data.dayBoundaryMode,
    convertToLayoverAtMinutes: data.convertToLayoverAtMinutes ?? null,
    layoverAccessorialChargeId: data.layoverAccessorialChargeId || null,
    notificationRequirement: data.notificationRequirement,
    notificationLeadMinutes: data.notificationLeadMinutes,
    notificationDeadlineMinutes: data.notificationDeadlineMinutes,
    unnotifiedBehavior: data.unnotifiedBehavior,
    autoSendNotice: data.autoSendNotice,
    sendDepartureSummary: data.sendDepartureSummary,
    requireApprovalOverAmount: decimalInput(data.requireApprovalOverAmount),
    autoApproveUnderAmount: decimalInput(data.autoApproveUnderAmount),
    currency: data.currency,
    comments: data.comments ?? "",
    version: data.version ?? null,
  };
}

export class DetentionService {
  public async desk(): Promise<DeskEntry[]> {
    const response = (await requestGraphQL({
      document: DetentionDeskDocument,
      operationName: "DetentionDesk",
      variables: {},
    })) as DetentionDeskQuery;

    return safeParse(deskEntryListSchema, response.detentionDesk, "Detention Desk");
  }

  public async getOccurrence(id: string): Promise<OccurrenceDetail> {
    const response = (await requestGraphQL({
      document: DetentionOccurrenceDetailDocument,
      operationName: "DetentionOccurrenceDetail",
      variables: { id },
    })) as DetentionOccurrenceDetailQuery;

    return safeParse(
      occurrenceDetailSchema,
      response.detentionOccurrence,
      "Detention Occurrence",
    );
  }

  public async byShipment(shipmentId: string): Promise<DetentionOccurrence[]> {
    const response = (await requestGraphQL({
      document: ShipmentDetentionDocument,
      operationName: "ShipmentDetention",
      variables: { shipmentId },
    })) as ShipmentDetentionQuery;

    return safeParse(
      z.array(detentionOccurrenceSchema),
      response.shipmentDetention,
      "Detention Occurrences",
    );
  }

  public async disputePacket(id: string): Promise<DisputePacket> {
    const response = (await requestGraphQL({
      document: DetentionDisputePacketDocument,
      operationName: "DetentionDisputePacket",
      variables: { occurrenceId: id },
    })) as DetentionDisputePacketQuery;

    return safeParse(disputePacketSchema, response.detentionDisputePacket, "Dispute Packet");
  }

  public async waive(
    id: string,
    payload: { reason: WaiverReason; note: string },
  ): Promise<DetentionOccurrence> {
    const response = (await requestGraphQL({
      document: WaiveDetentionOccurrenceDocument,
      operationName: "WaiveDetentionOccurrence",
      variables: {
        input: { occurrenceId: id, reason: payload.reason, note: payload.note },
      },
    })) as WaiveDetentionOccurrenceMutation;

    return safeParse(
      detentionOccurrenceSchema,
      response.waiveDetentionOccurrence,
      "Detention Occurrence",
    );
  }

  public async approve(id: string): Promise<DetentionOccurrence> {
    const response = (await requestGraphQL({
      document: ApproveDetentionOccurrenceDocument,
      operationName: "ApproveDetentionOccurrence",
      variables: { occurrenceId: id },
    })) as ApproveDetentionOccurrenceMutation;

    return safeParse(
      detentionOccurrenceSchema,
      response.approveDetentionOccurrence,
      "Detention Occurrence",
    );
  }

  public async dispute(
    id: string,
    payload: { note: string },
  ): Promise<DetentionOccurrence> {
    const response = (await requestGraphQL({
      document: DisputeDetentionOccurrenceDocument,
      operationName: "DisputeDetentionOccurrence",
      variables: { input: { occurrenceId: id, note: payload.note } },
    })) as DisputeDetentionOccurrenceMutation;

    return safeParse(
      detentionOccurrenceSchema,
      response.disputeDetentionOccurrence,
      "Detention Occurrence",
    );
  }

  public async sendNotice(id: string): Promise<DetentionOccurrence> {
    const response = (await requestGraphQL({
      document: SendDetentionNoticeDocument,
      operationName: "SendDetentionNotice",
      variables: { occurrenceId: id },
    })) as SendDetentionNoticeMutation;

    return safeParse(
      detentionOccurrenceSchema,
      response.sendDetentionNotice,
      "Detention Occurrence",
    );
  }
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
    const response = (await requestGraphQL({
      document: DetentionBacktestDocument,
      operationName: "DetentionBacktest",
      variables: {
        input: {
          policy: toDetentionPolicyInput(policy),
          from: params.from,
          to: params.to,
          assumeNoticeCompliance: params.assumeNoticeCompliance ?? null,
          driverPayRate: params.driverPayRate ?? null,
        },
      },
    })) as DetentionBacktestMutation;

    return safeParse(backtestResultSchema, response.detentionBacktest, "Detention Backtest");
  }

  public async facilities(params: {
    from: number;
    to: number;
    limit?: number;
  }): Promise<FacilityDetentionStat[]> {
    const response = (await requestGraphQL({
      document: DetentionFacilityStatsDocument,
      operationName: "DetentionFacilityStats",
      variables: {
        input: { from: params.from, to: params.to, limit: params.limit ?? null },
      },
    })) as DetentionFacilityStatsQuery;

    return safeParse(
      z.array(facilityDetentionStatSchema),
      response.detentionFacilityStats,
      "Facility Detention Stats",
    );
  }

  public async customers(params: {
    from: number;
    to: number;
    limit?: number;
  }): Promise<CustomerDetentionStat[]> {
    const response = (await requestGraphQL({
      document: DetentionCustomerStatsDocument,
      operationName: "DetentionCustomerStats",
      variables: {
        input: { from: params.from, to: params.to, limit: params.limit ?? null },
      },
    })) as DetentionCustomerStatsQuery;

    return safeParse(
      z.array(customerDetentionStatSchema),
      response.detentionCustomerStats,
      "Customer Detention Stats",
    );
  }

  public async waivers(params: {
    from: number;
    to: number;
  }): Promise<WaiverLeakageStat[]> {
    const response = (await requestGraphQL({
      document: DetentionWaiverStatsDocument,
      operationName: "DetentionWaiverStats",
      variables: { input: { from: params.from, to: params.to } },
    })) as DetentionWaiverStatsQuery;

    return safeParse(
      z.array(waiverLeakageStatSchema),
      response.detentionWaiverStats,
      "Detention Waiver Leakage",
    );
  }
}

export class DetentionPolicyService {
  public async getById(id: string): Promise<DetentionPolicy> {
    const response = (await requestGraphQL({
      document: DetentionPolicyDocument,
      operationName: "DetentionPolicy",
      variables: { id },
    })) as DetentionPolicyQuery;

    return safeParse(detentionPolicySchema, response.detentionPolicy, "Detention Policy");
  }

  public async create(data: DetentionPolicy): Promise<DetentionPolicy> {
    const response = (await requestGraphQL({
      document: CreateDetentionPolicyDocument,
      operationName: "CreateDetentionPolicy",
      variables: { input: toDetentionPolicyInput(data) },
    })) as CreateDetentionPolicyMutation;

    return safeParse(
      detentionPolicySchema,
      response.createDetentionPolicy,
      "Detention Policy",
    );
  }

  public async update(id: string, data: DetentionPolicy): Promise<DetentionPolicy> {
    const response = (await requestGraphQL({
      document: UpdateDetentionPolicyDocument,
      operationName: "UpdateDetentionPolicy",
      variables: { id, input: toDetentionPolicyInput(data) },
    })) as UpdateDetentionPolicyMutation;

    return safeParse(
      detentionPolicySchema,
      response.updateDetentionPolicy,
      "Detention Policy",
    );
  }

  public async delete(id: string): Promise<boolean> {
    const response = (await requestGraphQL({
      document: DeleteDetentionPolicyDocument,
      operationName: "DeleteDetentionPolicy",
      variables: { id },
    })) as DeleteDetentionPolicyMutation;

    return response.deleteDetentionPolicy;
  }

  /**
   * Runs the server's production calculator against a hypothetical stop so the
   * builder shows what the configured terms would actually charge. The math is
   * never re-derived on the client.
   */
  public async preview(
    policy: DetentionPolicy,
    scenario: PreviewScenario,
  ): Promise<PreviewResult> {
    const response = (await requestGraphQL({
      document: DetentionPolicyPreviewDocument,
      operationName: "DetentionPolicyPreview",
      variables: {
        policy: toDetentionPolicyInput(policy),
        scenario: {
          arrivedAt: scenario.arrivedAt,
          departedAt: scenario.departedAt ?? null,
          appointmentStart: scenario.appointmentStart ?? null,
          appointmentEnd: scenario.appointmentEnd ?? null,
          stopType: scenario.stopType as StopType,
          scheduleType: scenario.scheduleType as StopScheduleType,
          noticeSentAt: scenario.noticeSentAt ?? null,
          driverPayRate: scenario.driverPayRate ?? null,
        },
      },
    })) as DetentionPolicyPreviewQuery;

    return safeParse(
      previewResultSchema,
      response.detentionPolicyPreview,
      "Detention Preview",
    );
  }
}
