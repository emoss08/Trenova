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
import React from "react";
import { Link } from "react-router-dom";
import { LinksComponent } from "./nav-links";
import { useHeaderStore } from "@/stores/HeaderStore";

// Type Definitions
type PermissionType = string;

type MenuData = {
  menuKey: string;
  label: string;
  permission?: PermissionType;
  content?: React.ReactNode;
  link?: string;
};

type NavigationMenuItemProps = {
  data: MenuData;
  setMenuOpen: React.Dispatch<React.SetStateAction<string | undefined>>;
};

type SubLink = {
  label: string;
  link: string;
  permission?: string;
  description: string;
};

type MainLink = {
  label: string;
  link?: string;
  permission?: string;
  description: string;
  subLinks?: SubLink[];
};

type MainItem = {
  links: MainLink[];
};

/**
 * Check if user has access to the provided menu content.
 */
const userHasAccessToMenuContent = (
  content: React.ReactNode,
  userHasPermission: (permission: string) => boolean,
  isAdmin: boolean,
): boolean => {
  if (isAdmin) return true;

  if (React.isValidElement(content) && content.type === LinksComponent) {
    const linkData = content.props.linkData as MainItem[];

    return linkData.some((mainItem) =>
      mainItem.links.some((subItem) => {
        if (subItem.subLinks && !subItem.permission) {
          return subItem.subLinks.some(
            (link) => !link.permission || userHasPermission(link.permission),
          );
        }
        return !subItem.permission || userHasPermission(subItem.permission);
      }),
    );
  }
  return false;
};

const NavigationMenuItemWithPermission: React.FC<NavigationMenuItemProps> =
  React.memo(({ data, setMenuOpen }) => {
    const { userHasPermission, isAdmin } = useUserPermissions();

    if (data.permission && !userHasPermission(data.permission)) {
      return null;
    }

    if (!userHasAccessToMenuContent(data.content, userHasPermission, isAdmin)) {
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
      permission: "view_admin",
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
