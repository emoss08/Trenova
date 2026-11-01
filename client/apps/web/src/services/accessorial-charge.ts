import {
  CreateAccessorialChargeDocument,
  PatchAccessorialChargeDocument,
  UpdateAccessorialChargeDocument,
  type AccessorialChargeInput,
  type AccessorialChargePatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";

export function toAccessorialChargeInput(data: AccessorialCharge): AccessorialChargeInput {
  return {
    status: data.status,
    code: data.code,
    description: data.description,
    method: data.method,
    rateUnit: data.rateUnit ?? null,
    amount: String(data.amount),
    version: data.version,
  };
}

export class AccessorialChargeService {
  public async create(data: AccessorialCharge) {
    const response = await requestGraphQL({
      document: CreateAccessorialChargeDocument,
      operationName: "CreateAccessorialCharge",
      variables: { input: toAccessorialChargeInput(data) },
    });

    return response.createAccessorialCharge;
  }

  public async update(id: string, data: AccessorialCharge) {
    const response = await requestGraphQL({
      document: UpdateAccessorialChargeDocument,
      operationName: "UpdateAccessorialCharge",
      variables: { id, input: toAccessorialChargeInput(data) },
    });

    return response.updateAccessorialCharge;
  }

  public async patch(id: string, input: AccessorialChargePatchInput) {
    const response = await requestGraphQL({
      document: PatchAccessorialChargeDocument,
      operationName: "PatchAccessorialCharge",
      variables: { id, input },
    });

    return response.patchAccessorialCharge;
  }
}
