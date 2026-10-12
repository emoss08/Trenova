import { apiService } from "@/services/api";
import { createBatchLoader } from "@trenova/shared/lib/batch-loader";
import type { AvatarSummary } from "@trenova/shared/types/user";

/**
 * Every avatar on a screen asks for its person on its own; the loader collects the ids
 * asked for in one tick and fetches their gradient styles and availability together.
 */
const avatarLoader = createBatchLoader<string, AvatarSummary>(
  async (ids) => {
    const avatars = await apiService.userService.listAvatars(ids);
    return new Map(avatars.map((avatar) => [avatar.id, avatar]));
  },
  { maxBatchSize: 100 },
);

export async function loadUserAvatar(userId: string): Promise<AvatarSummary | null> {
  return (await avatarLoader.load(userId)) ?? null;
}
