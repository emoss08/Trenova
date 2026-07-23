import {
  EdiCommunicationProfileTableDocument,
  EdiInboundFileTableDocument,
  EdiMappingProfileTableDocument,
  EdiMessageTableDocument,
  EdiPartnerTableDocument,
  EdiTestCaseTableDocument,
  EdiTransferTableDocument,
  type EdiCommunicationProfileTableQueryVariables,
  type EdiInboundFileTableQueryVariables,
  type EdiMappingProfileTableQueryVariables,
  type EdiMessageTableQueryVariables,
  type EdiPartnerTableQueryVariables,
  type EdiTestCaseTableQueryVariables,
  type EdiTransferDirection,
  type EdiTransferTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type {
  EDICommunicationProfile,
  EDIInboundFile,
  EDIMappingProfile,
  EDIMessage,
  EDIPartner,
  EDITestCaseRow,
  EDITransfer,
} from "@/types/edi";

export const ediTableGraphQLConfigs = {
  partners: defineDataTableGraphQLConfig<EDIPartner, EdiPartnerTableQueryVariables>({
    document: EdiPartnerTableDocument,
    operationName: "EdiPartnerTable",
    connectionKey: "ediPartners",
  }),
  communicationProfiles: defineDataTableGraphQLConfig<
    EDICommunicationProfile,
    EdiCommunicationProfileTableQueryVariables
  >({
    document: EdiCommunicationProfileTableDocument,
    operationName: "EdiCommunicationProfileTable",
    connectionKey: "ediCommunicationProfiles",
  }),
  inboundTransfers: defineDataTableGraphQLConfig<EDITransfer, EdiTransferTableQueryVariables>({
    document: EdiTransferTableDocument,
    operationName: "EdiTransferTable",
    connectionKey: "ediTransfers",
    extraVariables: {
      direction: "Inbound" satisfies EdiTransferDirection,
    },
  }),
  outboundTransfers: defineDataTableGraphQLConfig<EDITransfer, EdiTransferTableQueryVariables>({
    document: EdiTransferTableDocument,
    operationName: "EdiTransferTable",
    connectionKey: "ediTransfers",
    extraVariables: {
      direction: "Outbound" satisfies EdiTransferDirection,
    },
  }),
  messages: defineDataTableGraphQLConfig<EDIMessage, EdiMessageTableQueryVariables>({
    document: EdiMessageTableDocument,
    operationName: "EdiMessageTable",
    connectionKey: "ediMessages",
  }),
  inboundFiles: defineDataTableGraphQLConfig<EDIInboundFile, EdiInboundFileTableQueryVariables>({
    document: EdiInboundFileTableDocument,
    operationName: "EdiInboundFileTable",
    connectionKey: "ediInboundFiles",
  }),
  mappingProfiles: defineDataTableGraphQLConfig<
    EDIMappingProfile,
    EdiMappingProfileTableQueryVariables
  >({
    document: EdiMappingProfileTableDocument,
    operationName: "EdiMappingProfileTable",
    connectionKey: "ediMappingProfiles",
  }),
  testCases: defineDataTableGraphQLConfig<EDITestCaseRow, EdiTestCaseTableQueryVariables>({
    document: EdiTestCaseTableDocument,
    operationName: "EdiTestCaseTable",
    connectionKey: "ediTestCases",
  }),
} as const;
