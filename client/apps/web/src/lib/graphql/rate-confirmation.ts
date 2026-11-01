import {
  ConfirmRateConfirmationDocument,
  GenerateRateConfirmationDocument,
  RateConfirmationsByMoveDocument,
  SendRateConfirmationDocument,
  VoidRateConfirmationDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export async function listRateConfirmationsByMoveGraphQL(
  moveId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: RateConfirmationsByMoveDocument,
    operationName: "RateConfirmationsByMove",
    variables: { moveId },
    signal: options?.signal,
  });
  return data.rateConfirmationsByMove;
}

export async function generateRateConfirmationGraphQL(moveId: string) {
  const data = await requestGraphQL({
    document: GenerateRateConfirmationDocument,
    operationName: "GenerateRateConfirmation",
    variables: { moveId },
  });
  return data.generateRateConfirmation;
}

export async function sendRateConfirmationGraphQL(id: string) {
  const data = await requestGraphQL({
    document: SendRateConfirmationDocument,
    operationName: "SendRateConfirmation",
    variables: { id },
  });
  return data.sendRateConfirmation;
}

export async function confirmRateConfirmationGraphQL(id: string, confirmedByName: string) {
  const data = await requestGraphQL({
    document: ConfirmRateConfirmationDocument,
    operationName: "ConfirmRateConfirmation",
    variables: { id, confirmedByName },
  });
  return data.confirmRateConfirmation;
}

export async function voidRateConfirmationGraphQL(id: string, reason: string) {
  const data = await requestGraphQL({
    document: VoidRateConfirmationDocument,
    operationName: "VoidRateConfirmation",
    variables: { id, reason },
  });
  return data.voidRateConfirmation;
}
