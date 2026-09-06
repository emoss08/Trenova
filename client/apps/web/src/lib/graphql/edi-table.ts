import {
  EdiCommunicationProfileTableDocument,
  EdiInboundFileTableDocument,
  EdiMappingProfileTableDocument,
  EdiMessageTableDocument,
  EdiPartnerTableDocument,
  EdiTestCaseTableDocument,
  EdiTransferTableDocument,
  type EdiTransferDirection,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow, DataTableRow } from "@trenova/shared/types/data-table";
import { ediMappingResolutionSchema, loadTenderPayloadSchema } from "@trenova/shared/types/edi";
import { z } from "zod";

const mappingSnapshotSchema = z.array(ediMappingResolutionSchema);

type EdiTransferNode = DataTableRow<typeof EdiTransferTableDocument, "ediTransfers">;

function parseTransferRow(node: EdiTransferNode) {
  return {
    ...node,
    tenderPayload: loadTenderPayloadSchema.parse(node.tenderPayload),
    mappingSnapshot: mappingSnapshotSchema.parse(node.mappingSnapshot ?? []),
  };
}

export const ediTableGraphQLConfigs = {
  partners: defineDataTableGraphQLConfig({
    document: EdiPartnerTableDocument,
    operationName: "EdiPartnerTable",
    connectionKey: "ediPartners",
  }),
  communicationProfiles: defineDataTableGraphQLConfig({
    document: EdiCommunicationProfileTableDocument,
    operationName: "EdiCommunicationProfileTable",
    connectionKey: "ediCommunicationProfiles",
  }),
  inboundTransfers: defineDataTableGraphQLConfig({
    document: EdiTransferTableDocument,
    operationName: "EdiTransferTable",
    connectionKey: "ediTransfers",
    extraVariables: {
      direction: "Inbound" satisfies EdiTransferDirection,
    },
    mapNode: parseTransferRow,
  }),
  outboundTransfers: defineDataTableGraphQLConfig({
    document: EdiTransferTableDocument,
    operationName: "EdiTransferTable",
    connectionKey: "ediTransfers",
    extraVariables: {
      direction: "Outbound" satisfies EdiTransferDirection,
    },
    mapNode: parseTransferRow,
  }),
  messages: defineDataTableGraphQLConfig({
    document: EdiMessageTableDocument,
    operationName: "EdiMessageTable",
    connectionKey: "ediMessages",
  }),
  inboundFiles: defineDataTableGraphQLConfig({
    document: EdiInboundFileTableDocument,
    operationName: "EdiInboundFileTable",
    connectionKey: "ediInboundFiles",
  }),
  mappingProfiles: defineDataTableGraphQLConfig({
    document: EdiMappingProfileTableDocument,
    operationName: "EdiMappingProfileTable",
    connectionKey: "ediMappingProfiles",
  }),
  testCases: defineDataTableGraphQLConfig({
    document: EdiTestCaseTableDocument,
    operationName: "EdiTestCaseTable",
    connectionKey: "ediTestCases",
  }),
} as const;

export type EDIMappingProfileRow = DataTableConfigRow<
  typeof ediTableGraphQLConfigs.mappingProfiles
>;
export type EDICommunicationProfileRow = DataTableConfigRow<
  typeof ediTableGraphQLConfigs.communicationProfiles
>;
export type EDITransferRow = DataTableConfigRow<typeof ediTableGraphQLConfigs.inboundTransfers>;
export type EDIMessageRow = DataTableConfigRow<typeof ediTableGraphQLConfigs.messages>;
export type EDIInboundFileRow = DataTableConfigRow<typeof ediTableGraphQLConfigs.inboundFiles>;
export type EDITestCaseTableRow = DataTableConfigRow<typeof ediTableGraphQLConfigs.testCases>;
