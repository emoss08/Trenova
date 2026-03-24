import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  updateStatusSchema,
  versionInfoSchema,
  type UpdateStatus,
  type VersionInfo,
} from "@/types/update";

export const updateService = {
  getVersion: async (): Promise<VersionInfo> => {
    const response = await api.get<VersionInfo>("/system/version");
    return safeParse(versionInfoSchema, response, "Version Info");
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
