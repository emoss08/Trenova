import { NavigationMenuLink } from "@/components/ui/navigation-menu";
import { useUserPermissions } from "@/context/user-permissions";
import { cn } from "@/lib/utils";
import React from "react";
import { Link, useLocation } from "react-router-dom";

type PermissionType = string;

export const ListItem = React.forwardRef<
  React.ElementRef<typeof Link>,
  React.ComponentPropsWithoutRef<typeof Link> & {
    permission?: PermissionType;
  }
>(({ className, title, children, permission, to, ...props }, ref) => {
  const { userHasPermission } = useUserPermissions();
  const location = useLocation();

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
          "block select-none space-y-1 rounded-md p-3 leading-none no-underline outline-none transition-colors hover:bg-accent/70 hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground",
          location.pathname === to && "bg-accent text-accent-foreground",
          className,
        )}
        {...props}
      >
        <div className="text-sm font-medium leading-none">{title}</div>
        <p className="text-muted-foreground line-clamp-2 text-xs leading-snug">
          {children}
        </p>
      </Link>
    </NavigationMenuLink>
  );
});
ListItem.displayName = "ListItem";
