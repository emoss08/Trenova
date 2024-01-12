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

import { cn } from "@/lib/utils";
import React from "react";
import { Link, useLocation } from "react-router-dom";
import { buttonVariants } from "../ui/button";

type SidebarLink = {
  href: string;
  title: string;
  icon?: React.ReactNode;
  group?: string;
};

interface SidebarNavProps extends React.HTMLAttributes<HTMLElement> {
  links: SidebarLink[];
}

export function SidebarNav({ className, links, ...props }: SidebarNavProps) {
  const location = useLocation();

  // Define the type for the accumulator
  type GroupedLinks = Record<string, SidebarLink[]>;

  // Group links by 'group' property
  const groupedLinks = links.reduce((acc: GroupedLinks, link) => {
    const groupName = link.group || "ungrouped"; // Use 'ungrouped' as a default group
    if (!acc[groupName]) {
      acc[groupName] = [];
    }
    acc[groupName].push(link);
    return acc;
  }, {} as GroupedLinks); // Initialize acc as an empty object of type GroupedLinks

  return (
    <div className="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
      <aside className="mb-1 md:w-56">
        <nav className={cn("lg:flex-col lg:space-y-2", className)} {...props}>
          {Object.entries(groupedLinks).map(([group, groupLinks]) => (
            <div key={group} className="space-y-2">
              {group !== "ungrouped" && (
                <h3 className="ml-4 select-none font-semibold">{group}</h3>
              )}
              <div className="space-y-1">
                {groupLinks.map((link) => (
                  <Link
                    key={link.title}
                    to={link.href}
                    className={cn(
                      buttonVariants({ variant: "ghost" }),
                      location.pathname === link.href
                        ? "bg-muted"
                        : "hover:bg-muted",
                      "group justify-start flex items-center ml-2",
                    )}
                  >
                    {link.icon && <span className="mr-2">{link.icon}</span>}
                    {link.title}
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </nav>
      </aside>
    </div>
  );
}
