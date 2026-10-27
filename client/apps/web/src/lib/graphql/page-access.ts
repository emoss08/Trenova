import {
  PageAccessDocument,
  PageAccessFieldsFragmentDoc,
  PageAccessPreviewDocument,
  PageAccessRequirementFieldsFragmentDoc,
  PageAccessScreensDocument,
  RoleResourceGuidesDocument,
  type PageAccessDecision,
  type PageAccessFieldsFragment,
  type PageAccessRequirementFieldsFragment,
  type PageAccessUse,
  type RoleResourceGuidesQuery,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import type { RoleDraft } from "@/lib/role-draft";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type { PageAccessDecision, PageAccessUse, RoleDraft };

export type PageAccessRequirement = PageAccessRequirementFieldsFragment;

export type PageAccessEndpoint = {
  method: string;
  path: string;
  requires: PageAccessRequirement[];
};

export type PageAccessKind = PageAccessFieldsFragment["kinds"][number];

/**
 * What a screen needs and what a person or a role would see on it: the
 * permissions its route and REST endpoints check, and each kind of record it
 * shows, with every field of a linked record it selects and whether that
 * field comes back.
 */
export type PageAccess = {
  path: string;
  name: string;
  operations: string[];
  requires: PageAccessRequirement[];
  endpoints: PageAccessEndpoint[];
  kinds: PageAccessKind[];
  withheld: number;
  blocked: boolean;
};

export type PageAccessScreen = { path: string; name: string };

export type RoleResourceGuide = RoleResourceGuidesQuery["roleResourceGuides"][number];

/** A role to preview a screen as: a saved one, or the one being edited. */
export type PageAccessSubject = { roleId: string } | { draft: RoleDraft };

function toPageAccess(
  masked: FragmentType<typeof PageAccessFieldsFragmentDoc> | null,
): PageAccess | null {
  if (!masked) {
    return null;
  }
  const access = getFragmentData(PageAccessFieldsFragmentDoc, masked);
  const requirements = (list: FragmentType<typeof PageAccessRequirementFieldsFragmentDoc>[]) =>
    getFragmentData(PageAccessRequirementFieldsFragmentDoc, list).map((entry) => ({ ...entry }));

  return {
    path: access.path,
    name: access.name,
    operations: [...access.operations],
    requires: requirements(access.requires),
    endpoints: access.endpoints.map((endpoint) => ({
      method: endpoint.method,
      path: endpoint.path,
      requires: requirements(endpoint.requires),
    })),
    kinds: access.kinds,
    withheld: access.withheld,
    blocked: access.blocked,
  };
}

/** The caller's own access to the screen at a location; null where no screen is. */
export async function fetchPageAccess(
  location: string,
  options?: RequestOptions,
): Promise<PageAccess | null> {
  const data = await requestGraphQL({
    document: PageAccessDocument,
    operationName: "PageAccess",
    variables: { location },
    signal: options?.signal,
  });

  return toPageAccess(data.pageAccess);
}

/** What a person holding only a role would need and see on the screen at a location. */
export async function fetchPageAccessPreview(
  location: string,
  subject: PageAccessSubject,
  options?: RequestOptions,
): Promise<PageAccess | null> {
  const data = await requestGraphQL({
    document: PageAccessPreviewDocument,
    operationName: "PageAccessPreview",
    variables: { input: { location, ...subject } },
    signal: options?.signal,
  });

  return toPageAccess(data.pageAccessPreview);
}

/** Every screen of the app, to choose one to preview. */
export async function fetchPageAccessScreens(
  options?: RequestOptions,
): Promise<PageAccessScreen[]> {
  const data = await requestGraphQL({
    document: PageAccessScreensDocument,
    operationName: "PageAccessScreens",
    variables: {},
    signal: options?.signal,
  });

  return data.pageAccessScreens.map((screen) => ({ path: screen.path, name: screen.name }));
}

/** The most resources one request words, as the server allows. */
export const MAX_GUIDE_RESOURCES = 64;

/** What granting and withholding each resource means, for a role reaching a sensitivity. */
export async function fetchRoleResourceGuides(
  resources: readonly string[],
  maxSensitivity: string,
  options?: RequestOptions,
): Promise<RoleResourceGuide[]> {
  const data = await requestGraphQL({
    document: RoleResourceGuidesDocument,
    operationName: "RoleResourceGuides",
    variables: {
      input: { resources: resources.slice(0, MAX_GUIDE_RESOURCES), maxSensitivity },
    },
    signal: options?.signal,
  });

  return data.roleResourceGuides;
}
