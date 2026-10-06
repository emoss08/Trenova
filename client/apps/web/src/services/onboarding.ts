import { api } from "@trenova/shared/lib/api";
import {
  onboardingStateSchema,
  type CompleteOnboardingRequest,
  type OnboardingState,
} from "@/types/onboarding";

export const onboardingService = {
  async get(signal?: AbortSignal): Promise<OnboardingState> {
    const response = await api.get<unknown>("/onboarding/", { signal });
    return onboardingStateSchema.parse(response ?? {});
  },

  async complete(request: CompleteOnboardingRequest): Promise<OnboardingState | null> {
    const response = await api.post<unknown>("/onboarding/complete/", request);
    const parsed = onboardingStateSchema.safeParse(response ?? {});
    return parsed.success && response ? parsed.data : null;
  },
};
