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
import {
  MenuData,
  NavigationMenuItemProps,
  calculatePosition,
  hasPermission,
  userHasAccessToContent,
} from "@/lib/navmenu";

// NavigationMenuItemWithPermission Component
const NavigationMenuItemWithPermission = React.memo(
  React.forwardRef<HTMLLIElement, NavigationMenuItemProps>(
    ({ data, setMenuOpen, setMenuPosition, menuItemRefs }, ref) => {
      const { userHasPermission, isAdmin } = useUserPermissions();

      // Handle mouse enter event
      const handleMouseEnter = () => {
        if (menuItemRefs.current[data.menuKey]) {
          const newPosition = calculatePosition(
            menuItemRefs.current[data.menuKey],
          );
          newPosition.left -= 200;

          setMenuPosition(newPosition);
          setMenuOpen(data.menuKey);
        }
      };

      // Render null if user does not have permission
      if (
        !hasPermission(data, userHasPermission) ||
        !userHasAccessToContent(data.content, userHasPermission, isAdmin)
      ) {
        return null;
      }

      // Render menu item with appropriate link or trigger
      return (
        <NavigationMenuItem ref={ref} onMouseEnter={handleMouseEnter}>
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
              <NavigationMenuTrigger
                onClick={() => setMenuOpen(data.menuKey)}
                onMouseEnter={handleMouseEnter}
              >
                {data.label}
              </NavigationMenuTrigger>
              <NavigationMenuContent>{data.content}</NavigationMenuContent>
            </>
          )}
        </NavigationMenuItem>
      );
    },
  ),
);

export function NavMenu() {
  const [menuOpen, setMenuOpen] = useHeaderStore.use("menuOpen");
  const [menuPosition, setMenuPosition] = React.useState({ left: 0, width: 0 });
  const menuItemRefs = React.useRef<Record<string, HTMLLIElement>>({});
  const navMenuRef = React.useRef<HTMLDivElement>(null);

  // Attach menu item ref
  const attachRef = React.useCallback(
    (key: string) => (element: HTMLLIElement | null) => {
      if (element) {
        menuItemRefs.current[key] = element;
      }
    },
    [],
  );

  // Handle clicks outside the menu to close it
  const handleClickOutside = (event: MouseEvent) => {
    if (
      navMenuRef.current &&
      !navMenuRef.current.contains(event.target as Node)
    ) {
      setMenuOpen(undefined);
    }
  };

  // Add and remove click outside listener
  React.useEffect(() => {
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Update menu position when menuOpen changes
  React.useEffect(() => {
    if (menuOpen && menuItemRefs.current[menuOpen]) {
      const position = calculatePosition(menuItemRefs.current[menuOpen]);
      // Adjust the left position here
      position.left -= 200; // Adjust the value as needed
      setMenuPosition(position);
    }
  }, [menuOpen, calculatePosition]);

  // Define menu items
  const menuItems: MenuData[] = [
    {
      menuKey: "dashboardMenu",
      label: "Dashboard",
      link: "/",
      permission: "view_dashbaord",
    },
    {
      menuKey: "billingMenu",
      label: "Billing & AR",
      content: <LinksComponent linkData={billingNavLinks} />,
    },
    {
      menuKey: "dispatchMenu",
      label: "Dispatch Management",
      content: <LinksComponent linkData={dispatchNavLinks} />,
    },
    {
      menuKey: "equipmentMenu",
      label: "Equipment Management",
      content: <LinksComponent linkData={equipmentNavLinks} />,
    },
    {
      menuKey: "shipmentMenu",
      label: "Shipment Management",
      content: <LinksComponent linkData={shipmentNavLinks} />,
    },
    {
      menuKey: "adminMenu",
      label: "Administrator",
      link: "/admin/dashboard/",
      permission: "view_admin_dashboard",
    },
  ];

  return (
    <div ref={navMenuRef}>
      {/* Hamburger Menu (visible on small screens) */}
      <button onClick={() => setMenuOpen(menuOpen)} className="md:hidden p-2">
        🍔
      </button>
      {/* Navigation Menu */}
      <NavigationMenu
        value={menuOpen}
        onValueChange={(newValue) => newValue && setMenuOpen(newValue)}
        onMouseLeave={() => setMenuOpen(undefined)}
        menuPosition={menuPosition}
        className={cn(
          menuOpen ? "block" : "hidden",
          "md:flex",
          "md:space-x-8",
          "lg:space-x-12",
          "xl:space-x-16",
        )}
      >
        <NavigationMenuList>
          {menuItems.map((item) => (
            <NavigationMenuItemWithPermission
              key={item.menuKey}
              data={item}
              setMenuPosition={setMenuPosition}
              setMenuOpen={setMenuOpen}
              ref={attachRef(item.menuKey)}
              menuItemRefs={menuItemRefs}
            />
          ))}
        </NavigationMenuList>
      </NavigationMenu>
    </div>
  );
}
