import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  CompleteOnboardingDocument,
  OnboardingStateDocument,
  OnboardingStateFieldsFragmentDoc,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import {
  onboardingStateSchema,
  type CompleteOnboardingRequest,
  type OnboardingState,
} from "@/types/onboarding";

export const onboardingService = {
  async get(signal?: AbortSignal): Promise<OnboardingState> {
    const response = await requestGraphQL({
      document: OnboardingStateDocument,
      operationName: "OnboardingState",
      signal,
    });
    return onboardingStateSchema.parse(
      getFragmentData(OnboardingStateFieldsFragmentDoc, response.onboardingState) ?? {},
    );
  },

  async complete(request: CompleteOnboardingRequest): Promise<OnboardingState | null> {
    const response = await requestGraphQL({
      document: CompleteOnboardingDocument,
      operationName: "CompleteOnboarding",
      variables: {
        input: {
          organization: {
            ...request.organization,
            scacCode: request.organization.scacCode ?? null,
            dotNumber: request.organization.dotNumber ?? null,
          },
          operationType: request.operationType,
          loadSampleData: request.loadSampleData,
        },
      },
    });
    const state = getFragmentData(OnboardingStateFieldsFragmentDoc, response.completeOnboarding);
    const parsed = onboardingStateSchema.safeParse(state ?? {});
    return parsed.success && state ? parsed.data : null;
  },
};
