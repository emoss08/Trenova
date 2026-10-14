import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import {
  AddAgentOrganizationProfileGuidanceDocument,
  AddMyAgentProfileGuidanceDocument,
  AgentOrganizationProfileDocument,
  AgentOrganizationProfileHistoryDocument,
  AgentProfileFieldsFragmentDoc,
  AgentProfileGuidanceFieldsFragmentDoc,
  AgentProfileVersionFieldsFragmentDoc,
  ApproveAgentOrganizationProfileGuidanceDocument,
  DismissAgentOrganizationProfileGuidanceDocument,
  MyAgentProfileDocument,
  RegenerateAgentOrganizationProfileDocument,
  RegenerateMyAgentProfileDocument,
  RetireAgentOrganizationProfileGuidanceDocument,
  RetireMyAgentProfileGuidanceDocument,
  type AgentProfileFieldsFragment,
  type AgentProfileGuidanceFieldsFragment,
  type AgentProfileVersionFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentProfileVersion = AgentProfileVersionFieldsFragment;
export type AgentProfileGuidance = AgentProfileGuidanceFieldsFragment;

/** A profile with its versions and guidance unmasked, as the panels read it. */
export type AgentProfile = Omit<
  AgentProfileFieldsFragment,
  "current" | "latestFailure" | "guidance"
> & {
  current: AgentProfileVersion | null;
  latestFailure: AgentProfileVersion | null;
  guidance: AgentProfileGuidance[];
};

/** Which profile a call is about: the organization's, or the signed-in person's own. */
export type AgentProfileOwner = "organization" | "mine";

/** The realtime resource a change to a profile or its guidance is announced on. */
export const AGENT_PROFILE_KEY = "agent-profile";

/** The most guidance one line may hold; the server refuses anything longer. */
export const MAX_PROFILE_GUIDANCE_CHARS = 500;

const HISTORY_LIMIT = 20;

type RequestOptions = { signal?: AbortSignal };

function unmask(profile: FragmentType<typeof AgentProfileFieldsFragmentDoc>): AgentProfile {
  const fields = getFragmentData(AgentProfileFieldsFragmentDoc, profile);

  return {
    ...fields,
    current: fields.current
      ? getFragmentData(AgentProfileVersionFieldsFragmentDoc, fields.current)
      : null,
    latestFailure: fields.latestFailure
      ? getFragmentData(AgentProfileVersionFieldsFragmentDoc, fields.latestFailure)
      : null,
    guidance: fields.guidance.map((line) =>
      getFragmentData(AgentProfileGuidanceFieldsFragmentDoc, line),
    ),
  };
}

export const agentProfileQueryKey = (owner: AgentProfileOwner) =>
  [AGENT_PROFILE_KEY, owner] as const;

export const agentProfileHistoryQueryKey = [AGENT_PROFILE_KEY, "organization", "history"] as const;

/**
 * What agents are told about the organization's business, or about the signed-in
 * person's own work. A profile not yet written comes back with no current version.
 */
export async function fetchAgentProfile(
  owner: AgentProfileOwner,
  options?: RequestOptions,
): Promise<AgentProfile> {
  if (owner === "mine") {
    const data = await requestGraphQL({
      document: MyAgentProfileDocument,
      operationName: "MyAgentProfile",
      variables: {},
      signal: options?.signal,
    });

    return unmask(data.myAgentProfile);
  }

  const data = await requestGraphQL({
    document: AgentOrganizationProfileDocument,
    operationName: "AgentOrganizationProfile",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.agentOrganizationProfile);
}

/** The organization profile's versions, newest first, failed attempts included. */
export async function fetchAgentProfileHistory(
  options?: RequestOptions,
): Promise<AgentProfileVersion[]> {
  const data = await requestGraphQL({
    document: AgentOrganizationProfileHistoryDocument,
    operationName: "AgentOrganizationProfileHistory",
    variables: { limit: HISTORY_LIMIT },
    signal: options?.signal,
  });

  return data.agentOrganizationProfileHistory.map((version) =>
    getFragmentData(AgentProfileVersionFieldsFragmentDoc, version),
  );
}

export async function addAgentProfileGuidance(
  owner: AgentProfileOwner,
  content: string,
): Promise<AgentProfileGuidance> {
  const input = { content: content.trim() };
  if (owner === "mine") {
    const data = await requestGraphQL({
      document: AddMyAgentProfileGuidanceDocument,
      operationName: "AddMyAgentProfileGuidance",
      variables: { input },
    });

    return getFragmentData(AgentProfileGuidanceFieldsFragmentDoc, data.addMyAgentProfileGuidance);
  }

  const data = await requestGraphQL({
    document: AddAgentOrganizationProfileGuidanceDocument,
    operationName: "AddAgentOrganizationProfileGuidance",
    variables: { input },
  });

  return getFragmentData(
    AgentProfileGuidanceFieldsFragmentDoc,
    data.addAgentOrganizationProfileGuidance,
  );
}

export async function retireAgentProfileGuidance(
  owner: AgentProfileOwner,
  id: string,
): Promise<AgentProfileGuidance> {
  if (owner === "mine") {
    const data = await requestGraphQL({
      document: RetireMyAgentProfileGuidanceDocument,
      operationName: "RetireMyAgentProfileGuidance",
      variables: { id },
    });

    return getFragmentData(
      AgentProfileGuidanceFieldsFragmentDoc,
      data.retireMyAgentProfileGuidance,
    );
  }

  const data = await requestGraphQL({
    document: RetireAgentOrganizationProfileGuidanceDocument,
    operationName: "RetireAgentOrganizationProfileGuidance",
    variables: { id },
  });

  return getFragmentData(
    AgentProfileGuidanceFieldsFragmentDoc,
    data.retireAgentOrganizationProfileGuidance,
  );
}

/**
 * An administrator's decision on a line someone offered about the organization:
 * approving keeps it and the profile is written again with it; dismissing closes
 * it before it ever shaped the profile. Only the organization's lines are offered.
 */
export async function approveAgentProfileGuidance(id: string): Promise<AgentProfileGuidance> {
  const data = await requestGraphQL({
    document: ApproveAgentOrganizationProfileGuidanceDocument,
    operationName: "ApproveAgentOrganizationProfileGuidance",
    variables: { id },
  });

  return getFragmentData(
    AgentProfileGuidanceFieldsFragmentDoc,
    data.approveAgentOrganizationProfileGuidance,
  );
}

export async function dismissAgentProfileGuidance(id: string): Promise<AgentProfileGuidance> {
  const data = await requestGraphQL({
    document: DismissAgentOrganizationProfileGuidanceDocument,
    operationName: "DismissAgentOrganizationProfileGuidance",
    variables: { id },
  });

  return getFragmentData(
    AgentProfileGuidanceFieldsFragmentDoc,
    data.dismissAgentOrganizationProfileGuidance,
  );
}

/** Asks for the profile to be written again now; it arrives a moment later. */
export async function regenerateAgentProfile(owner: AgentProfileOwner): Promise<void> {
  if (owner === "mine") {
    await requestGraphQL({
      document: RegenerateMyAgentProfileDocument,
      operationName: "RegenerateMyAgentProfile",
      variables: {},
    });

    return;
  }

  await requestGraphQL({
    document: RegenerateAgentOrganizationProfileDocument,
    operationName: "RegenerateAgentOrganizationProfile",
    variables: {},
  });
}
