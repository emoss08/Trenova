import {
  CarrierTrackingLinkDocument,
  IssueCarrierTrackingLinkDocument,
  LogCheckCallDocument,
  RevokeCarrierTrackingLinkDocument,
  ShipmentCheckCallsDocument,
  type CarrierTrackingLinkFieldsFragment,
  type CarrierTrackingLinkQuery,
  type CarrierTrackingLinkQueryVariables,
  type CheckCallFieldsFragment,
  type IssueCarrierTrackingLinkInput,
  type IssueCarrierTrackingLinkMutation,
  type IssueCarrierTrackingLinkMutationVariables,
  type LogCheckCallInput,
  type LogCheckCallMutation,
  type LogCheckCallMutationVariables,
  type RevokeCarrierTrackingLinkMutation,
  type RevokeCarrierTrackingLinkMutationVariables,
  type ShipmentCheckCallsQuery,
  type ShipmentCheckCallsQueryVariables,
  type TrackingEventOutcome,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type CheckCallRow = CheckCallFieldsFragment;
export type CarrierTrackingLinkRow = CarrierTrackingLinkFieldsFragment;

export type LoggedCheckCall = {
  checkCall: CheckCallRow;
  outcome: TrackingEventOutcome | null;
  outcomeReason: string | null;
};

export type IssuedCarrierTrackingLink = {
  link: CarrierTrackingLinkRow;
  url: string;
  emailed: boolean;
  emailError: string | null;
};

export async function listShipmentCheckCalls(
  shipmentId: string,
  options?: { signal?: AbortSignal },
): Promise<CheckCallRow[]> {
  const data = await requestGraphQL<ShipmentCheckCallsQuery, ShipmentCheckCallsQueryVariables>({
    document: ShipmentCheckCallsDocument,
    operationName: "ShipmentCheckCalls",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentCheckCalls as CheckCallRow[];
}

export async function getCarrierTrackingLink(
  shipmentMoveId: string,
  options?: { signal?: AbortSignal },
): Promise<CarrierTrackingLinkRow | null> {
  const data = await requestGraphQL<CarrierTrackingLinkQuery, CarrierTrackingLinkQueryVariables>({
    document: CarrierTrackingLinkDocument,
    operationName: "CarrierTrackingLink",
    variables: { shipmentMoveId },
    signal: options?.signal,
  });
  return (data.carrierTrackingLink as CarrierTrackingLinkRow | null | undefined) ?? null;
}

export async function logCheckCall(input: LogCheckCallInput): Promise<LoggedCheckCall> {
  const data = await requestGraphQL<LogCheckCallMutation, LogCheckCallMutationVariables>({
    document: LogCheckCallDocument,
    operationName: "LogCheckCall",
    variables: { input },
  });
  const payload = data.logCheckCall;
  return {
    checkCall: payload.checkCall as CheckCallRow,
    outcome: payload.outcome ?? null,
    outcomeReason: payload.outcomeReason ?? null,
  };
}

export async function issueCarrierTrackingLink(
  input: IssueCarrierTrackingLinkInput,
): Promise<IssuedCarrierTrackingLink> {
  const data = await requestGraphQL<
    IssueCarrierTrackingLinkMutation,
    IssueCarrierTrackingLinkMutationVariables
  >({
    document: IssueCarrierTrackingLinkDocument,
    operationName: "IssueCarrierTrackingLink",
    variables: { input },
  });
  const payload = data.issueCarrierTrackingLink;
  return {
    link: payload.link as CarrierTrackingLinkRow,
    url: payload.url,
    emailed: payload.emailed,
    emailError: payload.emailError ?? null,
  };
}

export async function revokeCarrierTrackingLink(shipmentMoveId: string): Promise<boolean> {
  const data = await requestGraphQL<
    RevokeCarrierTrackingLinkMutation,
    RevokeCarrierTrackingLinkMutationVariables
  >({
    document: RevokeCarrierTrackingLinkDocument,
    operationName: "RevokeCarrierTrackingLink",
    variables: { shipmentMoveId },
  });
  return data.revokeCarrierTrackingLink;
}
