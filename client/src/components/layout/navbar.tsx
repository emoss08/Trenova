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

import { Logo } from "@/components/layout/logo";
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuList,
  NavigationMenuTrigger,
  navigationMenuTriggerStyle,
} from "@/components/ui/navigation-menu";
import { useUserPermissions } from "@/context/user-permissions";
import {
  billingNavLinks,
  dispatchNavLinks,
  equipmentNavLinks,
  shipmentNavLinks,
} from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { useHeaderStore } from "@/stores/HeaderStore";
import React from "react";
import { Link } from "react-router-dom";
import { LinksComponent } from "./nav-links";

type Permission = string | undefined;

interface SubLink {
  label: string;
  link: string;
  permission?: Permission;
  description: string;
}

interface MainLink extends Omit<SubLink, "link" | "description"> {
  link?: string;
  subLinks?: SubLink[];
}

interface MainItem {
  links: MainLink[];
}

type MenuContent = React.ReactNode | MainItem[];

type MenuData = {
  menuKey: string;
  label: string;
  permission?: Permission;
  content?: React.ReactNode;
  link?: string;
};

type NavigationMenuItemProps = {
  data: MenuData;
  setMenuOpen: React.Dispatch<React.SetStateAction<string | undefined>>;
};

// Utility Functions
const hasPermission = (
  item: { permission?: Permission },
  userHasPermission: (permission: string) => boolean,
): boolean => !item.permission || userHasPermission(item.permission);

const userHasAccessToContent = (
  content: MenuContent,
  userHasPermission: (permission: string) => boolean,
  isAdmin: boolean,
): boolean => {
  if (isAdmin) return true;

  if (Array.isArray(content)) {
    return content.some((mainItem) =>
      mainItem.links.some(
        (link) =>
          hasPermission(link, userHasPermission) &&
          (!link.subLinks ||
            link.subLinks.some((subLink) =>
              hasPermission(subLink, userHasPermission),
            )),
      ),
    );
  }
  return true;
};

// NavigationMenuItemWithPermission Component
const NavigationMenuItemWithPermission: React.FC<NavigationMenuItemProps> =
  React.memo(({ data, setMenuOpen }) => {
    const { userHasPermission, isAdmin } = useUserPermissions();

    if (
      !hasPermission(data, userHasPermission) ||
      !userHasAccessToContent(data.content, userHasPermission, isAdmin)
    ) {
      return null;
    }

    return (
      <NavigationMenuItem>
        {data.link ? (
          <Link
            className={navigationMenuTriggerStyle()}
            to={data.link}
            onMouseEnter={() => setMenuOpen(undefined)}
          >
            {data.label}
          </Link>
        ) : (
          <>
            <NavigationMenuTrigger onClick={() => setMenuOpen(data.menuKey)}>
              {data.label}
            </NavigationMenuTrigger>
            <NavigationMenuContent>{data.content}</NavigationMenuContent>
          </>
        )}
      </NavigationMenuItem>
    );
  });

export function NavMenu() {
  const [menuOpen, setMenuOpen] = useHeaderStore.use("menuOpen");

  // Define menu items
  const menuItems: MenuData[] = [
    {
      menuKey: "BillingMenu",
      label: "Billing & AR",
      content: <LinksComponent linkData={billingNavLinks} />,
    },
    {
      menuKey: "DispatchMenu",
      label: "Dispatch Management",
      content: <LinksComponent linkData={dispatchNavLinks} />,
    },
    {
      menuKey: "EquipmentMenu",
      label: "Equipment Management",
      content: <LinksComponent linkData={equipmentNavLinks} />,
    },
    {
      menuKey: "ShipmentMenu",
      label: "Shipment Management",
      content: <LinksComponent linkData={shipmentNavLinks} />,
    },
    {
      menuKey: "AdminMenu",
      label: "Administrator",
      link: "/admin",
      permission: "admin.view_admin_dashboard",
    },
  ];

  return (
    <div>
      {/* Hamburger Menu (visible on small screens) */}
      <button onClick={() => setMenuOpen(menuOpen)} className="md:hidden p-2">
        🍔
      </button>

      {/* Navigation Menu */}
      <NavigationMenu
        value={menuOpen}
        onValueChange={(newValue) => newValue && setMenuOpen(newValue)}
        onMouseLeave={() => setMenuOpen(undefined)}
        className={cn(
          menuOpen ? "block" : "hidden", // Show/Hide based on state
          "md:flex", // Always flex on medium screens and above
          "md:space-x-8",
          "lg:space-x-12",
          "xl:space-x-16",
        )}
      >
        <NavigationMenuList>
          <Logo />
          {menuItems.map((item) => (
            <NavigationMenuItemWithPermission
              key={item.menuKey}
              data={item}
              setMenuOpen={setMenuOpen}
            />
          ))}
        </NavigationMenuList>
      </NavigationMenu>
    </div>
  );
}
