import {
  OrganizationSettingsDocument,
  UpdateOrganizationSettingsDocument,
  type OrganizationInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import {
  organizationSettingsSchema,
  type OrganizationSettings,
} from "@/types/organization";

type OrganizationSettingsResponse = {
  organization: unknown;
};

type UpdateOrganizationSettingsResponse = {
  updateOrganization: unknown;
};

export async function getOrganizationSettingsGraphQL(
  organizationId: OrganizationSettings["id"],
): Promise<OrganizationSettings> {
  const data = await requestGraphQL<OrganizationSettingsResponse>({
    document: OrganizationSettingsDocument,
    operationName: "OrganizationSettings",
    variables: {
      id: organizationId,
      includeState: true,
      includeBu: false,
    },
  });

  return safeParse(
    organizationSettingsSchema,
    data.organization,
    "OrganizationSettings",
  );
}

export async function updateOrganizationSettingsGraphQL(
  organizationId: OrganizationSettings["id"],
  values: OrganizationSettings,
): Promise<OrganizationSettings> {
  const data = await requestGraphQL<UpdateOrganizationSettingsResponse>({
    document: UpdateOrganizationSettingsDocument,
    operationName: "UpdateOrganizationSettings",
    variables: {
      id: organizationId,
      input: toOrganizationInput(values),
    },
  });

  return safeParse(
    organizationSettingsSchema,
    data.updateOrganization,
    "OrganizationSettings",
  );
}

function toOrganizationInput(values: OrganizationSettings): OrganizationInput {
  return {
    version: values.version,
    name: values.name,
    loginSlug: values.loginSlug ?? null,
    scacCode: values.scacCode,
    dotNumber: values.dotNumber,
    logoUrl: values.logoUrl ?? null,
    bucketName: values.bucketName ?? null,
    addressLine1: values.addressLine1,
    addressLine2: values.addressLine2 ?? null,
    city: values.city,
    stateId: values.stateId,
    postalCode: values.postalCode,
    timezone: values.timezone,
    taxId: values.taxId ?? null,
  };
}
