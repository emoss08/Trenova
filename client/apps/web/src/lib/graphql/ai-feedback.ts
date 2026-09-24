import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  ClearMyAiFeedbackDocument,
  MyAiFeedbackDocument,
  MyAiFeedbackFieldsFragmentDoc,
  SetMyAiFeedbackDocument,
  type AiFeedbackReason,
  type AiFeedbackTargetType,
  type MyAiFeedbackFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AiFeedback = MyAiFeedbackFieldsFragment;
export type { AiFeedbackReason, AiFeedbackTargetType };

/** A thumbs up is 1 and a thumbs down is -1, as the server stores them. */
export type AiFeedbackRating = 1 | -1;

/** What a rating is about: the record, and the part of it when it has parts. */
export type AiFeedbackTarget = {
  targetType: AiFeedbackTargetType;
  targetId: string;
  targetPart?: string;
};

export type AiFeedbackSubmission = {
  rating: AiFeedbackRating;
  reasons: AiFeedbackReason[];
  comment: string;
};

/** The most targets one read may name, matching the server's limit. */
export const AI_FEEDBACK_BATCH_LIMIT = 200;

/** One stable string per target, so a rating is found by what it is about. */
export function aiFeedbackTargetKey(target: AiFeedbackTarget): string {
  return `${target.targetType}:${target.targetId}:${target.targetPart ?? ""}`;
}

function targetInput(target: AiFeedbackTarget) {
  return {
    targetType: target.targetType,
    targetId: target.targetId,
    targetPart: target.targetPart ?? "",
  };
}

export async function fetchMyAiFeedback(
  targets: readonly AiFeedbackTarget[],
  options?: { signal?: AbortSignal },
): Promise<AiFeedback[]> {
  if (targets.length === 0) {
    return [];
  }

  const data = await requestGraphQL({
    document: MyAiFeedbackDocument,
    operationName: "MyAIFeedback",
    variables: { input: { targets: targets.map(targetInput) } },
    signal: options?.signal,
  });

  return data.myAIFeedback.map((entry) => getFragmentData(MyAiFeedbackFieldsFragmentDoc, entry));
}

export async function setMyAiFeedback(
  target: AiFeedbackTarget,
  submission: AiFeedbackSubmission,
): Promise<AiFeedback> {
  const data = await requestGraphQL({
    document: SetMyAiFeedbackDocument,
    operationName: "SetMyAIFeedback",
    variables: {
      input: {
        ...targetInput(target),
        rating: submission.rating,
        reasons: submission.reasons,
        comment: submission.comment,
      },
    },
  });

  return getFragmentData(MyAiFeedbackFieldsFragmentDoc, data.setMyAIFeedback);
}

export async function clearMyAiFeedback(target: AiFeedbackTarget): Promise<boolean> {
  const data = await requestGraphQL({
    document: ClearMyAiFeedbackDocument,
    operationName: "ClearMyAIFeedback",
    variables: { input: targetInput(target) },
  });

  return data.clearMyAIFeedback;
}
