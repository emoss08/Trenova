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

import { cn } from "@/lib/utils";
import React from "react";
import { Link, useLocation } from "react-router-dom";
import { buttonVariants } from "../ui/button";

interface SidebarNavProps extends React.HTMLAttributes<HTMLElement> {
  links: {
    href: string;
    title: string;
    icon?: React.ReactNode;
  }[];
}

export function SidebarNav({ className, links, ...props }: SidebarNavProps) {
  const location = useLocation();

  return (
    <div className="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
      <aside className="-mx-4 w-52">
        <nav
          className={cn(
            "flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-2",
            className,
          )}
          {...props}
        >
          {links.map((link) => (
            <Link
              key={link.title}
              to={link.href}
              className={cn(
                buttonVariants({ variant: "ghost" }),
                location.pathname === link.href
                  ? "bg-muted hover:bg-muted"
                  : "hover:bg-muted space-y-1",
                "group justify-start flex items-center", // Add flex and items-center classes
              )}
            >
              {link.icon && <span className="mr-2">{link.icon}</span>}{" "}
              {link.title}
            </Link>
          ))}
        </nav>
      </aside>
    </div>
  );
}
