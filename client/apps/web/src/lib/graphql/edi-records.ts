import type { EdiMappingProfileItemInput } from "@trenova/graphql/generated/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  ediInboundFileSchema,
  ediMessageInspectionSchema,
  ediMessageSchema,
  ediTransferSchema,
  type EDIMappingProfileItem,
} from "@trenova/shared/types/edi";

type GraphQLRecord = Record<string, unknown>;

/** What a message's detail carries beside the message row: its raw X12 and findings. */
type MessageExtras = {
  rawX12?: string | null;
  diagnosticCount?: number;
  diagnostics?: readonly unknown[];
};

function messageShape(raw: unknown, extras?: MessageExtras): GraphQLRecord {
  const diagnostics = (extras?.diagnostics ?? []) as GraphQLRecord[];
  const message = raw as GraphQLRecord;
  return {
    ...message,
    standard: "X12",
    rawX12: extras?.rawX12 ?? "",
    payloadSnapshot: message.payloadSnapshot ?? null,
    diagnosticCount: extras?.diagnosticCount ?? diagnostics.length,
    validationErrors: diagnostics.map((diagnostic) => ({
      ...diagnostic,
      elementPosition: diagnostic.elementPosition ?? undefined,
    })),
  };
}

/** A message as the message panel and card read it, from the GraphQL message detail. */
export function toEDIMessage(raw: unknown, extras?: MessageExtras) {
  return safeParse(ediMessageSchema, messageShape(raw, extras), "EDIMessage");
}

/** A message's segment-by-segment reading, with the message in the shape its card reads. */
export function toEDIMessageInspection(raw: unknown) {
  const inspection = raw as GraphQLRecord & { message: GraphQLRecord };
  return safeParse(
    ediMessageInspectionSchema,
    {
      ...inspection,
      message: messageShape(inspection.message, {
        rawX12: inspection.message.rawX12 as string | null | undefined,
      }),
    },
    "EDIMessageInspection",
  );
}

/**
 * An inbound file as the file panel and card read it: its row, its raw content, the
 * profile it came through and the transaction sets parsed from it (the first 50).
 */
export function toEDIInboundFile(raw: unknown) {
  const detail = raw as {
    file: GraphQLRecord;
    communicationProfileName?: string | null;
    messages: readonly unknown[];
  };
  return safeParse(
    ediInboundFileSchema,
    {
      ...detail.file,
      communicationProfile: detail.communicationProfileName
        ? { name: detail.communicationProfileName }
        : null,
      messages: detail.messages.map((message) => messageShape(message)),
    },
    "EDIInboundFile",
  );
}

export function toEDITransfer(raw: unknown) {
  return safeParse(ediTransferSchema, raw, "EDITransfer");
}

/** The mapping entries a form holds, as the GraphQL input takes them. */
export function toMappingItemInputs(
  items: readonly EDIMappingProfileItem[],
): EdiMappingProfileItemInput[] {
  return items.map((item) => ({
    id: item.id || undefined,
    entityType: item.entityType,
    sourceId: item.sourceId,
    sourceLabel: item.sourceLabel ?? undefined,
    targetId: item.targetId,
    targetLabel: item.targetLabel ?? undefined,
  }));
}
