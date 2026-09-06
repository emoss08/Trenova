import {
  CreateOrgHolidayDocument,
  DeleteOrgHolidayDocument,
  OrgHolidaysDocument,
  UpdateOrgHolidayDocument,
  type CreateOrgHolidayMutation,
  type CreateOrgHolidayMutationVariables,
  type DeleteOrgHolidayMutation,
  type DeleteOrgHolidayMutationVariables,
  type OrgHolidayFieldsFragment,
  type OrgHolidayInput,
  type OrgHolidaysQuery,
  type OrgHolidaysQueryVariables,
  type UpdateOrgHolidayMutation,
  type UpdateOrgHolidayMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type OrgHolidayRow = OrgHolidayFieldsFragment;

export const ORG_HOLIDAYS_KEY = "org-holidays";

export async function fetchOrgHolidays(
  year: number,
  options?: { signal?: AbortSignal },
): Promise<OrgHolidayRow[]> {
  const data = await requestGraphQL<OrgHolidaysQuery, OrgHolidaysQueryVariables>({
    document: OrgHolidaysDocument,
    operationName: "OrgHolidays",
    variables: { year },
    signal: options?.signal,
  });
  return data.orgHolidays as OrgHolidayRow[];
}

export async function createOrgHoliday(input: OrgHolidayInput): Promise<OrgHolidayRow> {
  const data = await requestGraphQL<CreateOrgHolidayMutation, CreateOrgHolidayMutationVariables>({
    document: CreateOrgHolidayDocument,
    operationName: "CreateOrgHoliday",
    variables: { input },
  });
  return data.createOrgHoliday as OrgHolidayRow;
}

export async function updateOrgHoliday(id: string, input: OrgHolidayInput): Promise<OrgHolidayRow> {
  const data = await requestGraphQL<UpdateOrgHolidayMutation, UpdateOrgHolidayMutationVariables>({
    document: UpdateOrgHolidayDocument,
    operationName: "UpdateOrgHoliday",
    variables: { id, input },
  });
  return data.updateOrgHoliday as OrgHolidayRow;
}

export async function deleteOrgHoliday(id: string): Promise<boolean> {
  const data = await requestGraphQL<DeleteOrgHolidayMutation, DeleteOrgHolidayMutationVariables>({
    document: DeleteOrgHolidayDocument,
    operationName: "DeleteOrgHoliday",
    variables: { id },
  });
  return data.deleteOrgHoliday;
}
