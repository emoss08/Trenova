import { api } from "@trenova/shared/lib/api";
import {
  SELF_HOSTED_PUBLIC_CONFIG,
  publicConfigSchema,
  type PublicConfig,
} from "@trenova/shared/types/platform";

export const PUBLIC_CONFIG_ENDPOINT = "/system/public-config";

export const platformService = {
  async getPublicConfig(signal?: AbortSignal): Promise<PublicConfig> {
    try {
      const response = await api.get<unknown>(PUBLIC_CONFIG_ENDPOINT, { signal });
      const parsed = publicConfigSchema.safeParse(response ?? {});
      if (parsed.success) {
        return parsed.data;
      }
      console.warn("[platform] public config did not match the expected shape", parsed.error);
      return SELF_HOSTED_PUBLIC_CONFIG;
    } catch (error) {
      if (signal?.aborted) {
        throw error;
      }
      return SELF_HOSTED_PUBLIC_CONFIG;
    }
  },
};
