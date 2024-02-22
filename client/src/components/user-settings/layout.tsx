/*
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

import { SidebarLink } from "@/types/sidebar-nav";
import {
  faBellRing,
  faGear,
  faRightLeft,
  faSatelliteDish,
  faShieldHalved,
  faStarHalf,
  faUser,
} from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Suspense } from "react";
import { ComponentLoader } from "../ui/component-loader";
import { SidebarNav } from "./sidebar-nav";

const links: SidebarLink[] = [
  {
    href: "/account/settings/",
    title: "User Settings",
    icon: (
      <FontAwesomeIcon
        icon={faUser}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: false,
  },
  {
    href: "/account/settings/preferences/",
    title: "Preferences",
    icon: (
      <FontAwesomeIcon
        icon={faStarHalf}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: false,
  },
  {
    href: "#",
    title: "Notifications",
    icon: (
      <FontAwesomeIcon
        icon={faBellRing}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: true,
  },
  {
    href: "#",
    title: "API Keys",
    icon: (
      <FontAwesomeIcon
        icon={faRightLeft}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: true,
  },
  {
    href: "#",
    title: "Connections",
    icon: (
      <FontAwesomeIcon
        icon={faSatelliteDish}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: true,
  },
  {
    href: "#",
    title: "Privacy",
    icon: (
      <FontAwesomeIcon
        icon={faShieldHalved}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: true,
  },
  {
    href: "#",
    title: "Advanced",
    icon: (
      <FontAwesomeIcon
        icon={faGear}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    disabled: true,
  },
];

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
      <SidebarNav links={links} />
      <main className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <Suspense fallback={<ComponentLoader />}>{children}</Suspense>
        </div>
      </main>
    </div>
  );
}
