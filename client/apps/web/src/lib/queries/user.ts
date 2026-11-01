import { userService } from "@trenova/shared/services/user";
import { loadUserAvatar } from "@/lib/user-avatars";
import { notificationPreferencesService } from "@trenova/shared/services/notification-preferences";

import type { User } from "@trenova/shared/types/user";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const userOrganization = createQueryKeys("userOrganization", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async () => userService.getUserOrganizations(),
  }),
});

export const user = createQueryKeys("user", {
  all: null,
  detail: (id: User["id"]) => ({
    queryKey: ["detail", id],
    queryFn: async () => userService.get(id),
  }),
  account: (id: string) => ({
    queryKey: ["account", id],
    queryFn: async ({ signal }) => {
      const { fetchUserDetail } = await import("@/lib/graphql/users");
      return fetchUserDetail(id, { signal });
    },
  }),
  profilePicture: (id: User["id"], variant: "thumbnail" | "full" = "thumbnail") => ({
    queryKey: ["profile-picture", id, variant],
    queryFn: async () => (id ? userService.getProfilePictureURL(id, variant) : null),
  }),
  avatar: (id: string) => ({
    queryKey: ["avatar", id],
    queryFn: async () => loadUserAvatar(id),
  }),
  mySessions: () => ({
    queryKey: ["my-sessions"],
    queryFn: async () => userService.listMySessions(),
  }),
  mySignIns: () => ({
    queryKey: ["my-sign-ins"],
    queryFn: async () => userService.listMySignIns(5),
  }),
  myNotificationPreferences: () => ({
    queryKey: ["my-notification-preferences"],
    queryFn: async () => notificationPreferencesService.get(),
  }),
});
