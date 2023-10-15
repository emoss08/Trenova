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

import { NavigationMenuLink } from "@/components/ui/navigation-menu";
import { useUserPermissions } from "@/context/user-permissions";
import { cn } from "@/lib/utils";
import React from "react";
import { Link } from "react-router-dom";

type PermissionType = string;

export const ListItem = React.forwardRef<
  React.ElementRef<typeof Link>,
  React.ComponentPropsWithoutRef<typeof Link> & {
    permission?: PermissionType;
  }
>(({ className, title, children, permission, to, ...props }, ref) => {
  const { userHasPermission } = useUserPermissions();

  // If the ListItem has a permission and the user doesn't have it, return null
  if (permission && !userHasPermission(permission)) {
    return null;
  }

  return (
    <NavigationMenuLink asChild>
      <Link
        ref={ref}
        to={to}
        className={cn(
          "block select-none space-y-1 rounded-md p-3 leading-none no-underline outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground",
          className,
        )}
        {...props}
      >
        <div className="text-sm font-medium leading-none">{title}</div>
        <p className="line-clamp-2 text-sm leading-snug text-muted-foreground">
          {children}
        </p>
      </Link>
    </NavigationMenuLink>
  );
});
ListItem.displayName = "ListItem";
