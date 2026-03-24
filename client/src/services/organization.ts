import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  organizationLogoUrlResponseSchema,
  organizationSettingsSchema,
  type OrganizationSettings,
} from "@/types/organization";

export class OrganizationService {
  readonly base_url = "/organizations";

  public async getByID(
    organizationId: string,
    opts: { includeState?: boolean; includeBu?: boolean } = {},
  ) {
    const query = new URLSearchParams();
    query.set("includeState", String(opts.includeState ?? true));
    query.set("includeBu", String(opts.includeBu ?? false));

    const response = await api.get<OrganizationSettings>(
      `${this.base_url}/${organizationId}?${query.toString()}`,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async update(organizationId: string, data: OrganizationSettings) {
    const response = await api.put<OrganizationSettings>(
      `${this.base_url}/${organizationId}`,
      data,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async uploadLogo(organizationId: string, file: File) {
    const formData = new FormData();
    formData.append("file", file);

    const response = await api.upload<OrganizationSettings>(
      `${this.base_url}/${organizationId}/logo`,
      formData,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }

  public async getLogoURL(organizationId: string) {
    const response = await api.get<{ url: string }>(
      `${this.base_url}/${organizationId}/logo`,
    );

    return (await safeParse(organizationLogoUrlResponseSchema, response, "OrganizationLogoUrl")).url;
  }

  public async deleteLogo(organizationId: string) {
    const response = await api.delete<OrganizationSettings>(
      `${this.base_url}/${organizationId}/logo`,
    );

    return safeParse(organizationSettingsSchema, response, "OrganizationSettings");
  }
}
