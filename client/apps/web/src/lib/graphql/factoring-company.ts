import {
  CreateFactoringCompanyDocument,
  UpdateFactoringCompanyDocument,
  type FactoringCompanyInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { FactoringCompany } from "@/types/factoring-company";

export function toFactoringCompanyInput(values: FactoringCompany): FactoringCompanyInput {
  return {
    status: values.status,
    code: values.code,
    name: values.name,
    addressLine1: values.addressLine1,
    addressLine2: values.addressLine2 || null,
    city: values.city,
    stateId: values.stateId,
    postalCode: values.postalCode,
    email: values.email || null,
    phoneNumber: values.phoneNumber || null,
    remittanceInstructions: values.remittanceInstructions || null,
    noticeOfAssignment: values.noticeOfAssignment || null,
    nonRecourse: values.nonRecourse,
    notes: values.notes || null,
  };
}

export async function createFactoringCompany(values: FactoringCompany) {
  const data = await requestGraphQL({
    document: CreateFactoringCompanyDocument,
    operationName: "CreateFactoringCompany",
    variables: { input: toFactoringCompanyInput(values) },
  });

  return data.createFactoringCompany;
}

export async function updateFactoringCompany(id: string, values: FactoringCompany) {
  const data = await requestGraphQL({
    document: UpdateFactoringCompanyDocument,
    operationName: "UpdateFactoringCompany",
    variables: {
      input: {
        id,
        version: values.version ?? 0,
        factoringCompany: toFactoringCompanyInput(values),
      },
    },
  });

  return data.updateFactoringCompany;
}
