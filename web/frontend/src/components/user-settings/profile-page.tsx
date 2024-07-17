/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { User } from "@/types/accounts";
import { LinkGroupProps } from "@/types/sidebar-nav";
import { Suspense, lazy, useState } from "react";
import { ComponentLoader } from "../ui/component-loader";
import { ModalAsideMenu } from "./sidebar-nav";

const PersonalInformation = lazy(
  () => import("@/components/user-settings/personal-information-form"),
);

const PreferenceComponent = lazy(
  () => import("@/components/user-settings/user-preferences"),
);

const ChangePasswordForm = lazy(
  () => import("@/components/user-settings/change-password-form"),
);

export default function UserProfile({ user }: { user: User }) {
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
