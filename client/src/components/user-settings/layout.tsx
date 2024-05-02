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
          <Suspense fallback={<ComponentLoader className="h-[60vh]" />}>
            {children}
          </Suspense>
        </div>
      </main>
    </div>
  );
}
