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
import { debounce } from "lodash-es";
import React from "react";
import { Link, useLocation } from "react-router-dom";
import { buttonVariants } from "../ui/button";
import { ScrollArea } from "../ui/scroll-area";

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
  const [isScrolled, setIsScrolled] = React.useState(false);
  const scrollThreshold = 80;

  const handleScroll = React.useMemo(
    () =>
      debounce(() => {
        setIsScrolled(window.scrollY > scrollThreshold);
      }, 30),
    [scrollThreshold],
  );

  React.useEffect(() => {
    window.addEventListener("scroll", handleScroll);
    return () => {
      handleScroll.cancel(); // Ensure debounce is cancelled on unmount
      window.removeEventListener("scroll", handleScroll);
    };
  }, [handleScroll]);

  const groupedLinks = React.useMemo(() => {
    type GroupedLinks = Record<string, SidebarLink[]>;
    return links.reduce((acc: GroupedLinks, link) => {
      const groupName = link.group || "ungrouped";
      acc[groupName] = acc[groupName] || [];
      acc[groupName].push(link);
      return acc;
    }, {} as GroupedLinks);
  }, [links]);

  return (
    <aside
      className={cn(
        "transition-spacing fixed top-14 z-30 -ml-2 hidden h-[calc(100vh-10rem)] w-full shrink-0 duration-500 md:sticky md:block",
        isScrolled ? "pt-10" : "",
      )}
    >
      <ScrollArea className="bg-card text-card-foreground size-full rounded-lg border p-3">
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
                        ? "bg-muted [&_svg]:text-foreground"
                        : "hover:bg-muted",
                      "group justify-start flex items-center mx-2",
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
      </ScrollArea>
    </aside>
  );
}
