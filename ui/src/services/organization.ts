import { http } from "@/lib/http-client";
import { type Organization } from "@/types/organization";
import { type LimitOffsetResponse } from "@/types/server";

type GetOrgByIdOptions = {
  orgId: string;
  includeState?: boolean;
  includeBu?: boolean;
};

export async function getOrgById({
  orgId,
  includeState = false,
  includeBu = false,
}: GetOrgByIdOptions) {
  return http.get<Organization>(`/organizations/${orgId}`, {
    params: {
      includeState: includeState.toString(),
      includeBu: includeBu.toString(),
    },
  });
}

export async function listOrganizations() {
  return http.get<LimitOffsetResponse<Organization>>("/organizations/me");
}
