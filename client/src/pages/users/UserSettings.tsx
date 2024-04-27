import SettingsLayout from "@/components/user-settings/layout";
import { useUserStore } from "@/stores/AuthStore";
import { User } from "@/types/accounts";
import { lazy } from "react";

const UserProfilePage = lazy(
  () => import("@/components/user-settings/profile-page"),
);

export default function UserSettings() {
  const userData = useUserStore.get("user");

  return (
    <SettingsLayout>
      {userData && <UserProfilePage user={userData as User} />}
    </SettingsLayout>
  );
}
