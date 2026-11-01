import { apiService } from "@/services/api";
import { getUserSummary } from "@/lib/user-api";
import { loadUserAvatar } from "@/lib/user-avatars";
import { notificationPreferencesService } from "@trenova/shared/services/notification-preferences";

import { createQueryKeys } from "@lukemorales/query-key-factory";

export const userOrganization = createQueryKeys("userOrganization", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async () => apiService.userService.getUserOrganizations(),
  }),
});

export const user = createQueryKeys("user", {
  all: null,
  detail: (id: string) => ({
    queryKey: ["detail", id],
    queryFn: async ({ signal }) => getUserSummary(id, { signal }),
  }),
  account: (id: string) => ({
    queryKey: ["account", id],
    queryFn: async ({ signal }) => {
      const { fetchUserDetail } = await import("@/lib/graphql/users");
      return fetchUserDetail(id, { signal });
    },
  }),
  profilePicture: (id: string | undefined, variant: "thumbnail" | "full" = "thumbnail") => ({
    queryKey: ["profile-picture", id, variant],
    queryFn: async () => (id ? apiService.userService.getProfilePictureURL(id, variant) : null),
  }),
  avatar: (id: string) => ({
    queryKey: ["avatar", id],
    queryFn: async () => loadUserAvatar(id),
  }),
  mySessions: () => ({
    queryKey: ["my-sessions"],
    queryFn: async ({ signal }) => apiService.userService.listMySessions({ signal }),
  }),
  mySignIns: () => ({
    queryKey: ["my-sign-ins"],
    queryFn: async ({ signal }) => apiService.userService.listMySignIns(5, { signal }),
  }),
  myNotificationPreferences: () => ({
    queryKey: ["my-notification-preferences"],
    queryFn: async () => notificationPreferencesService.get(),
  }),
});
