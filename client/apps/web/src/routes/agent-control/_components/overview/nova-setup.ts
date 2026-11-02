import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AIProviderPlatform } from "@/types/ai-provider";
import { canRunNova } from "../providers/provider-model";

type NovaProviderFacts = Pick<
  AIProviderRow,
  "id" | "name" | "kind" | "platform" | "enabled" | "nova" | "priority" | "version"
>;

/**
 * What stands between Nova and running, if anything:
 *
 * - `ready`: a provider runs Nova and is on.
 * - `paused`: the provider chosen for Nova is turned off.
 * - `choose`: no provider runs Nova; `provider` is the one to run it on, the first on
 *   in routing order that can, else the first that can.
 * - `connect`: no provider can run Nova, so one has to be added.
 */
export type NovaSetup<P extends NovaProviderFacts = NovaProviderFacts> =
  | { state: "ready"; provider: P }
  | { state: "paused"; provider: P }
  | { state: "choose"; provider: P }
  | { state: "connect" };

export function novaSetup<P extends NovaProviderFacts>(
  providers: readonly P[],
  novaPlatforms: readonly AIProviderPlatform[],
): NovaSetup<P> {
  const current = providers.find((provider) => provider.nova);
  if (current) {
    return current.enabled
      ? { state: "ready", provider: current }
      : { state: "paused", provider: current };
  }

  const eligible = providers
    .filter((provider) => canRunNova(provider, novaPlatforms))
    .sort((a, b) => a.priority - b.priority);
  const candidate = eligible.find((provider) => provider.enabled) ?? eligible[0];

  return candidate ? { state: "choose", provider: candidate } : { state: "connect" };
}
