/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { useBreadcrumbs } from "@/hooks/use-breadcrumbs";
import React from "react";
import { useNavigation } from "react-router";
import { NavActions } from "./nav-actions";

export function Header() {
  const navigation = useNavigation();
  const breadcrumbs = useBreadcrumbs();
  const isLoading = navigation.state === "loading";

  return (
    <header className="flex h-14 w-full shrink-0 items-center gap-2 bg-background/95 backdrop-blur-sm supports-backdrop-filter:bg-background/60 dark:border-border">
      <div className="flex flex-1 items-center gap-2 px-3">
        <SidebarTrigger />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink className={isLoading ? "opacity-50" : ""}>
                Home
              </BreadcrumbLink>
            </BreadcrumbItem>
            {breadcrumbs.length > 0 && <BreadcrumbSeparator />}
            {breadcrumbs.map((crumb, index) => (
              <React.Fragment key={crumb.id}>
                <BreadcrumbItem>
                  {index < breadcrumbs.length - 1 ? (
                    <BreadcrumbLink className={isLoading ? "opacity-50" : ""}>
                      {crumb.crumb}
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage className="line-clamp-1">
                      {crumb.crumb}
                    </BreadcrumbPage>
                  )}
                </BreadcrumbItem>
                {index < breadcrumbs.length - 1 && <BreadcrumbSeparator />}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div className="ml-auto px-3">
        <NavActions />
      </div>
    </header>
  );
}
