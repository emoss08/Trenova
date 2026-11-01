import {
  BulkUpdateCarrierStatusDocument,
  CarrierRecordDocument,
  CreateCarrierDocument,
  PatchCarrierDocument,
  UpdateCarrierDocument,
  type CarrierInput,
  type CarrierPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  carrierSchema,
  type BulkUpdateCarrierStatusRequest,
  type Carrier,
} from "@trenova/shared/types/carrier";

export function toCarrierInput(data: Carrier): CarrierInput {
  return {
    stateId: data.stateId ?? null,
    remitStateId: data.remitStateId ?? null,
    status: data.status,
    code: data.code,
    name: data.name,
    dbaName: data.dbaName ?? null,
    carrierType: data.carrierType,
    dotNumber: data.dotNumber ?? null,
    mcNumber: data.mcNumber ?? null,
    scac: data.scac ?? null,
    complianceStatus: data.complianceStatus,
    safetyRating: data.safetyRating,
    qualifiedAt: data.qualifiedAt ?? null,
    disqualifiedReason: data.disqualifiedReason ?? null,
    taxId: data.taxId ?? null,
    taxIdType: data.taxIdType ?? null,
    w9OnFile: data.w9OnFile,
    is1099Eligible: data.is1099Eligible,
    paymentMethod: data.paymentMethod,
    paymentTermDays: data.paymentTermDays,
    remitToName: data.remitToName ?? null,
    remitAddressLine1: data.remitAddressLine1 ?? null,
    remitAddressLine2: data.remitAddressLine2 ?? null,
    remitCity: data.remitCity ?? null,
    remitPostalCode: data.remitPostalCode ?? null,
    addressLine1: data.addressLine1 ?? null,
    addressLine2: data.addressLine2 ?? null,
    city: data.city ?? null,
    postalCode: data.postalCode ?? null,
    phone: data.phone ?? null,
    email: data.email ?? null,
    externalId: data.externalId ?? null,
    notes: data.notes ?? null,
    version: data.version,
    contacts: data.contacts.map((contact) => ({
      id: contact.id || null,
      name: contact.name,
      title: contact.title ?? null,
      email: contact.email ?? null,
      phone: contact.phone ?? null,
      isPrimary: contact.isPrimary,
      receivesRateConfirmations: contact.receivesRateConfirmations,
      version: contact.version,
    })),
    insurancePolicies: data.insurancePolicies.map((policy) => ({
      id: policy.id || null,
      policyType: policy.policyType,
      policyNumber: policy.policyNumber,
      providerName: policy.providerName,
      coverageAmount: policy.coverageAmount == null ? "" : String(policy.coverageAmount),
      effectiveDate: policy.effectiveDate,
      expirationDate: policy.expirationDate,
      isVerified: policy.isVerified,
      version: policy.version,
    })),
  };
}

export class CarrierService {
  public async create(data: Carrier) {
    const response = await requestGraphQL({
      document: CreateCarrierDocument,
      operationName: "CreateCarrier",
      variables: { input: toCarrierInput(data) },
    });

    return response.createCarrier;
  }

  public async update(id: string, data: Carrier) {
    const response = await requestGraphQL({
      document: UpdateCarrierDocument,
      operationName: "UpdateCarrier",
      variables: { id, input: toCarrierInput(data) },
    });

    return response.updateCarrier;
  }

  public async bulkUpdateStatus(request: BulkUpdateCarrierStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateCarrierStatusDocument,
      operationName: "BulkUpdateCarrierStatus",
      variables: { input: request },
    });

    return response.bulkUpdateCarrierStatus;
  }

  /** The carrier as its form edits it, with its contacts and insurance policies. */
  public async getById(carrierId: string, options?: { signal?: AbortSignal }): Promise<Carrier> {
    const response = await requestGraphQL({
      document: CarrierRecordDocument,
      operationName: "CarrierRecord",
      variables: { id: carrierId },
      signal: options?.signal,
    });
    if (!response.carrier) {
      throw new Error("Carrier not found");
    }

    return safeParse(carrierSchema, response.carrier, "Carrier");
  }

  public async patch(carrierId: string, input: CarrierPatchInput) {
    const response = await requestGraphQL({
      document: PatchCarrierDocument,
      operationName: "PatchCarrier",
      variables: { id: carrierId, input },
    });

    return response.patchCarrier;
  }
}
