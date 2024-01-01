/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import {
  GearIcon,
  LinkBreak2Icon,
  PersonIcon,
  RocketIcon,
  TargetIcon,
} from "@radix-ui/react-icons";
import { ArrowRightLeftIcon, RouterIcon } from "lucide-react";
import { Suspense } from "react";
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "./sidebar-nav";

const links = [
  {
    href: "/account/settings/",
    title: "User Settings",
    icon: (
      <PersonIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "/account/settings/preferences/",
    title: "Preferences",
    icon: (
      <TargetIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "#",
    title: "Notifications",
    icon: (
      <RouterIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "#",
    title: "API Keys",
    icon: (
      <ArrowRightLeftIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "#",
    title: "Connections",
    icon: (
      <RocketIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "#",
    title: "Privacy",
    icon: (
      <LinkBreak2Icon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
  {
    href: "#",
    title: "Advanced",
    icon: (
      <GearIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
  },
];

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex bg-background-primary justify-center">
      <SidebarNav links={links} />
      <div className="flex-1 max-w-4xl mx-12 w-full">
        <Suspense fallback={<Skeleton className="h-[1000px] w-full" />}>
          {children}
        </Suspense>
      </div>
    </div>
  );
}
