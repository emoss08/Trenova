import SettingsLayout from "@/components/user-settings/layout";
import { useAuthenticatedUser } from "@/hooks/useQueries";
import { User } from "@/types/accounts";
import { lazy } from "react";

const UserProfilePage = lazy(
  () => import("@/components/user-settings/profile-page"),
);

export default function UserSettings() {
  const { data } = useAuthenticatedUser();

  return (
    <SettingsLayout>
      {data && <UserProfilePage user={data as User} />}
    </SettingsLayout>
  );
}
