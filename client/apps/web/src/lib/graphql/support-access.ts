import {
  CurrentSupportSessionDocument,
  DataTablePageInfoFieldsFragmentDoc,
  GrantSupportAccessDocument,
  RevokeSupportAccessDocument,
  SupportAccessActionFieldsFragmentDoc,
  SupportAccessActionsDocument,
  SupportAccessGrantFieldsFragmentDoc,
  SupportAccessGrantsDocument,
  SupportAccessSettingsDocument,
  type CurrentSupportSessionQuery,
  type GrantSupportAccessInput,
  type SupportAccessActionFieldsFragment,
  type SupportAccessGrantFieldsFragment,
  type SupportAccessSettingsQuery,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type SupportAccessGrant = SupportAccessGrantFieldsFragment;
export type SupportAccessAction = SupportAccessActionFieldsFragment;
export type SupportAccessSettings = SupportAccessSettingsQuery["supportAccessSettings"];
export type SupportSession = NonNullable<CurrentSupportSessionQuery["currentSupportSession"]>;
export type SupportAccessScope = SupportAccessGrant["scope"];

export type Page<T> = { items: T[]; nextCursor: string | null };

const PAGE_SIZE = 20;

/** Whether grants can be made on this instance and what an administrator may choose. */
export async function fetchSupportAccessSettings(options?: {
  signal?: AbortSignal;
}): Promise<SupportAccessSettings> {
  const data = await requestGraphQL({
    document: SupportAccessSettingsDocument,
    operationName: "SupportAccessSettings",
    signal: options?.signal,
  });
  return data.supportAccessSettings;
}

/** Support grants, live and past, newest first. */
export async function fetchSupportAccessGrants(
  after: string | null,
  options?: { signal?: AbortSignal },
): Promise<Page<SupportAccessGrant>> {
  const data = await requestGraphQL({
    document: SupportAccessGrantsDocument,
    operationName: "SupportAccessGrants",
    variables: {
      input: { first: PAGE_SIZE, after, sort: [{ field: "createdAt", direction: "desc" }] },
    },
    signal: options?.signal,
  });
  const connection = data.supportAccessGrants;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);
  return {
    items: connection.edges.map((edge) =>
      getFragmentData(SupportAccessGrantFieldsFragmentDoc, edge.node),
    ),
    nextCursor: pageInfo.hasNextPage ? (pageInfo.endCursor ?? null) : null,
  };
}

/**
 * What was done under one grant. The audit log credits every entry a support session wrote to
 * the grant itself, so the grant's id is the principal of exactly its own entries.
 */
export async function fetchSupportAccessActions(
  grantId: string,
  after: string | null,
  options?: { signal?: AbortSignal },
): Promise<Page<SupportAccessAction>> {
  const data = await requestGraphQL({
    document: SupportAccessActionsDocument,
    operationName: "SupportAccessActions",
    variables: {
      input: {
        first: PAGE_SIZE,
        after,
        fieldFilters: [{ field: "principalId", operator: "eq", value: grantId }],
        sort: [{ field: "timestamp", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });
  const connection = data.auditEntries;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);
  return {
    items: connection.edges.map((edge) =>
      getFragmentData(SupportAccessActionFieldsFragmentDoc, edge.node),
    ),
    nextCursor: pageInfo.hasNextPage ? (pageInfo.endCursor ?? null) : null,
  };
}

export async function grantSupportAccess(
  input: GrantSupportAccessInput,
): Promise<SupportAccessGrant> {
  const data = await requestGraphQL({
    document: GrantSupportAccessDocument,
    operationName: "GrantSupportAccess",
    variables: { input },
  });
  return getFragmentData(SupportAccessGrantFieldsFragmentDoc, data.grantSupportAccess);
}

export async function revokeSupportAccess(id: string, reason: string): Promise<SupportAccessGrant> {
  const data = await requestGraphQL({
    document: RevokeSupportAccessDocument,
    operationName: "RevokeSupportAccess",
    variables: { input: { id, reason } },
  });
  return getFragmentData(SupportAccessGrantFieldsFragmentDoc, data.revokeSupportAccess);
}

/** The caller's own session when Trenova support opened it, otherwise null. */
export async function fetchCurrentSupportSession(options?: {
  signal?: AbortSignal;
}): Promise<SupportSession | null> {
  const data = await requestGraphQL({
    document: CurrentSupportSessionDocument,
    operationName: "CurrentSupportSession",
    signal: options?.signal,
  });
  return data.currentSupportSession ?? null;
}
