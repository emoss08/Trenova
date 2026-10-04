import { platformService } from "@trenova/shared/services/platform";
import {
  SELF_HOSTED_PUBLIC_CONFIG,
  isCloudPlatform,
  isSignupAvailable,
  type PublicConfig,
} from "@trenova/shared/types/platform";
import { queryOptions, useQuery } from "@tanstack/react-query";

const PUBLIC_CONFIG_STALE_MS = 5 * 60 * 1000;

export const publicConfigQueryOptions = queryOptions({
  queryKey: ["system", "public-config"] as const,
  queryFn: async ({ signal }) => platformService.getPublicConfig(signal),
  staleTime: PUBLIC_CONFIG_STALE_MS,
  gcTime: Number.POSITIVE_INFINITY,
  retry: false,
  refetchOnWindowFocus: false,
});

export type PublicConfigState = {
  config: Readonly<PublicConfig>;
  isLoading: boolean;
  isCloud: boolean;
  signupAvailable: boolean;
};

export function usePublicConfig(): PublicConfigState {
  const query = useQuery(publicConfigQueryOptions);
  const config = query.data ?? SELF_HOSTED_PUBLIC_CONFIG;

  return {
    config,
    isLoading: query.isPending,
    isCloud: isCloudPlatform(config),
    signupAvailable: isSignupAvailable(config),
  };
}
