import { User } from "@/types/accounts";
import { Suspense, lazy, useState } from "react";
import { ComponentLoader } from "../ui/component-loader";
import { ModalAsideMenu } from "./sidebar-nav";
import { LinkGroupProps } from "@/types/sidebar-nav";

const PreferenceComponent = lazy(
  () => import("@/components/user-settings/user-preferences"),
);

const PersonalInformation = lazy(
  () => import("@/components/user-settings/personal-information-form"),
);

const ChangePasswordForm = lazy(
  () => import("@/components/user-settings/change-password-form"),
);

export default function UserProfileF({ user }: { user: User }) {
  const [activeTab, setActiveTab] = useState("personal-information");

  const linkGroups: LinkGroupProps[] = [
    {
      title: "Account Settings",
      links: [
        {
          key: "personal-information",
          href: "#personal-information",
          title: "Personal Information",
          component: <PersonalInformation user={user} />,
        },
        {
          key: "change-password",
          href: "#change-password",
          title: "Change Password",
          component: <ChangePasswordForm />,
        },
      ],
    },
    {
      title: "Preferences",
      links: [
        {
          key: "preferences",
          href: "#preferences",
          title: "Preferences",
          component: <PreferenceComponent />,
        },
        {
          key: "notifications",
          href: "#notifications",
          title: "Notifications",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "API and Connections",
      links: [
        {
          key: "api-keys",
          href: "#api-keys",
          title: "API Keys",
          component: <div>Coming soon</div>,
        },
        {
          key: "connections",
          href: "#connections",
          title: "Connections",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "Privacy and Security",
      links: [
        {
          key: "privacy",
          href: "#privacy",
          title: "Privacy",
          component: <div>Coming soon</div>,
        },
      ],
    },
  ];

  const activeComponent = linkGroups
    .flatMap((group) => group.links)
    .find((link) => link.key === activeTab)?.component;

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
      <ModalAsideMenu
        heading="Settings"
        linkGroups={linkGroups}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />
      <main className="relative max-h-[600px] lg:gap-10">
        <div className="mx-auto min-w-0">
          <Suspense fallback={<ComponentLoader className="h-[30vh]" />}>
            {activeComponent}
          </Suspense>
        </div>
      </main>
    </div>
  );
}
