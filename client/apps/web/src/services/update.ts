import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  networkPulseSchema,
  updateStatusSchema,
  versionInfoSchema,
  type NetworkPulse,
  type UpdateStatus,
  type VersionInfo,
} from "@/types/update";

export const updateService = {
  getVersion: async (): Promise<VersionInfo> => {
    const response = await api.get<VersionInfo>("/system/version");
    return safeParse(versionInfoSchema, response, "Version Info");
  },

  // Public, and 404s unless the operator has turned system.networkPulse on. Callers on
  // the sign-in screen must treat a failure as "nothing to show", not as an error.
  getNetworkPulse: async (): Promise<NetworkPulse> => {
    const response = await api.get<NetworkPulse>("/system/network-pulse");
    return safeParse(networkPulseSchema, response, "Network Pulse");
  },

  getUpdateStatus: async (): Promise<UpdateStatus> => {
    const response = await api.get<UpdateStatus>("/system/update-status");
    return safeParse(updateStatusSchema, response, "Update Status");
  },

  checkForUpdates: async (): Promise<UpdateStatus> => {
    const response = await api.post<UpdateStatus>("/system/check-updates");
    return safeParse(updateStatusSchema, response, "Update Status");
  },
};
