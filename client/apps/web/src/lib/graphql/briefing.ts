import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  BriefingFieldsFragmentDoc,
  MarkBriefingReadDocument,
  RegenerateBriefingDocument,
  TodaysBriefingDocument,
  type BriefingFieldsFragment,
  type BriefingRoleKey,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type Briefing = BriefingFieldsFragment;
export type BriefingSection = Briefing["sections"][number];
export type { BriefingRoleKey };

type RequestOptions = { signal?: AbortSignal };

/**
 * This morning's page, or null when it has not been written yet.
 *
 * Null is an ordinary answer rather than an error: the briefing is written on
 * a schedule, so before that hour there is simply nothing to read, and a
 * client that treated it as a failure would show an error every night.
 */
export async function fetchTodaysBriefing(
  roleKey?: BriefingRoleKey,
  options?: RequestOptions,
): Promise<Briefing | null> {
  const data = await requestGraphQL({
    document: TodaysBriefingDocument,
    operationName: "TodaysBriefing",
    variables: { input: { roleKey: roleKey ?? null, briefingDate: null } },
    signal: options?.signal,
  });

  return data.todaysBriefing
    ? getFragmentData(BriefingFieldsFragmentDoc, data.todaysBriefing)
    : null;
}

export async function markBriefingRead(id: string): Promise<Briefing> {
  const data = await requestGraphQL({
    document: MarkBriefingReadDocument,
    operationName: "MarkBriefingRead",
    variables: { id },
  });

  return getFragmentData(BriefingFieldsFragmentDoc, data.markBriefingRead);
}

export async function regenerateBriefing(roleKey?: BriefingRoleKey): Promise<Briefing> {
  const data = await requestGraphQL({
    document: RegenerateBriefingDocument,
    operationName: "RegenerateBriefing",
    variables: { input: { roleKey: roleKey ?? null, briefingDate: null } },
  });

  return getFragmentData(BriefingFieldsFragmentDoc, data.regenerateBriefing);
}
