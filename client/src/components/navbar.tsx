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

import { OrganizationLogo } from "@/components/layout/Navbar/_partials/OrganizationLogo";
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
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
import React, { useState } from "react";
import { Link } from "react-router-dom";
import { LinksComponent } from "./nav-links";

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
  // menuOpen: string | undefined;
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

const userHasAccessToMenuContent = (
  content: React.ReactNode,
  userHasPermission: (permission: string) => boolean,
  isAdmin: boolean,
): boolean => {
  // If the user is an admin, immediately grant access
  if (isAdmin) return true;

  // Check if content is a valid React Element and of type LinksComponent
  if (React.isValidElement(content) && content.type === LinksComponent) {
    const linkData = content.props.linkData as MainItem[];

    return linkData.some((mainItem) => {
      return mainItem.links.some((subItem) => {
        if (subItem.subLinks && !subItem.permission) {
          return subItem.subLinks.some(
            (link) => !link.permission || userHasPermission(link.permission),
          );
        } else {
          return !subItem.permission || userHasPermission(subItem.permission);
        }
      });
    });
  }
  return false;
};

const NavigationMenuItemWithPermission: React.FC<NavigationMenuItemProps> = ({
  data,
  setMenuOpen,
}) => {
  const { userHasPermission, isAdmin } = useUserPermissions();

  // Check for permissions and return null if not allowed
  if (data.permission && !userHasPermission(data.permission)) {
    return null;
  }

  // Check if the user has access to the menu content
  const hasAccess = userHasAccessToMenuContent(
    data.content,
    userHasPermission,
    isAdmin,
  );
  if (!hasAccess) {
    return null;
  }

  if (data.link) {
    console.info("data.link", data.link);
    return (
      <NavigationMenuItem>
        <Link to={data.link} onMouseEnter={() => setMenuOpen(undefined)}>
          <NavigationMenuLink className={navigationMenuTriggerStyle()}>
            {data.label}
          </NavigationMenuLink>
        </Link>
      </NavigationMenuItem>
    );
  }
  return (
    <NavigationMenuItem>
      <NavigationMenuTrigger onClick={() => setMenuOpen(data.menuKey)}>
        {data.label}
      </NavigationMenuTrigger>
      <NavigationMenuContent>{data.content}</NavigationMenuContent>
    </NavigationMenuItem>
  );
};

export function NavMenu() {
  const [menuOpen, setMenuOpen] = useState<string | undefined>();

  const handleValueChange = (newValue: string) => {
    if (newValue !== "") {
      setMenuOpen(newValue);
    }
  };

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
    <NavigationMenu
      value={menuOpen}
      onValueChange={handleValueChange}
      onMouseLeave={() => setMenuOpen(undefined)} // Close the menu on mouse leave
    >
      <NavigationMenuList>
        {/* Organization Logo */}
        <OrganizationLogo />

        {/* Navigation Menu Items */}
        {menuItems.map((item) => (
          <NavigationMenuItemWithPermission
            key={item.menuKey}
            data={item}
            setMenuOpen={setMenuOpen}
          />
        ))}
      </NavigationMenuList>
    </NavigationMenu>
  );
}
