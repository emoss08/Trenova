import SettingsLayout from "@/components/user-settings/layout";
import { lazy } from "react";

const UserPreferencesPage = lazy(
  () => import("@/components/user-settings/preference-page"),
);

export default function UserPreferences() {
  return (
    <SettingsLayout>
      <UserPreferencesPage />
    </SettingsLayout>
  );
}
