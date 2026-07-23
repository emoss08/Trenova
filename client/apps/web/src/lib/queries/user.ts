import { apiService } from "@/services/api";

import type { User } from "@/types/user";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const userOrganization = createQueryKeys("userOrganization", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async () => apiService.userService.getUserOrganizations(),
  }),
});

export const user = createQueryKeys("user", {
  all: null,
  detail: (id: User["id"]) => ({
    queryKey: ["detail", id],
    queryFn: async () => apiService.userService.get(id),
  }),
  profilePicture: (id: User["id"], variant: "thumbnail" | "full" = "thumbnail") => ({
    queryKey: ["profile-picture", id, variant],
    queryFn: async () => (id ? apiService.userService.getProfilePictureURL(id, variant) : null),
  }),
});
