import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { networkPulseSchema, type NetworkPulse } from "../types/network-pulse";

export const networkPulseService = {
  // Public, and 404s unless the operator has turned system.networkPulse on. Callers on
  // the sign-in screen must treat a failure as "nothing to show", not as an error.
  get: async (): Promise<NetworkPulse> => {
    const response = await api.get<NetworkPulse>("/system/network-pulse");
    return safeParse(networkPulseSchema, response, "Network Pulse");
  },
};
