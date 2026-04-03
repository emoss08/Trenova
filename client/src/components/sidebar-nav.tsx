import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import { usePermissionStore } from "@/stores/permission-store";
import type { OperationType } from "@/types/permission";
import React, { useMemo } from "react";
import { Link, useLocation } from "react-router";
import { BetaTag } from "./beta-tag";
import { ScrollArea } from "./ui/scroll-area";
import { Separator } from "./ui/separator";

export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
  adminOnly?: boolean;
  platformAdminOnly?: boolean;
  includeBetaTag?: boolean;
  resource?: string;
  requiredOperation?: OperationType;
};

type SidebarNavProps = React.HTMLAttributes<HTMLElement> & {
  links: SidebarLink[];
};

export function SidebarNav({ links, className, ...props }: SidebarNavProps) {
  const location = useLocation();
  const manifest = usePermissionStore((state) => state.manifest);

  const visibleLinks = useMemo(
    () =>
      links.filter((link) => {
        if (link.platformAdminOnly && !manifest?.isPlatformAdmin) {
          return false;
        }

        return true;
      }),
    [links, manifest?.isPlatformAdmin],
  );

  const groupedLinks = useMemo(() => {
    type GroupedLinks = Record<string, SidebarLink[]>;

    return visibleLinks.reduce((acc: GroupedLinks, link) => {
      const groupName = link.group || "ungrouped";
      acc[groupName] = acc[groupName] || [];
      acc[groupName].push(link);
      return acc;
    }, {} as GroupedLinks);
  }, [visibleLinks]);

  return (
    <SidebarNavOuter>
      <ScrollArea className="h-full border-r bg-background p-3 text-card-foreground">
        <SidebarNavInner className={cn("lg:flex-col", className)} {...props}>
          {Object.entries(groupedLinks).map(([group, groupLinks], index, array) => (
            <div key={group} className="space-y-2">
              {group !== "ungrouped" && (
                <h3 className="text-xs font-semibold text-muted-foreground uppercase select-none">
                  {group}
                </h3>
              )}
              <div>
                {groupLinks.map((link) => {
                  const isActive = location.pathname === link.href;
                  return (
                    <Link
                      key={link.title}
                      to={link.href}
                      className={cn(
                        buttonVariants({ variant: "ghost" }),
                        isActive
                          ? "bg-muted dark:bg-primary/10"
                          : "hover:bg-muted dark:hover:bg-primary/10",
                        link.disabled && "pointer-events-none opacity-50",
                        "group mb-1 flex items-center justify-start font-normal text-xs",
                      )}
                    >
                      {link.title}
                      {link.includeBetaTag && <BetaTag />}
                    </Link>
                  );
                })}
                {index !== array.length - 1 && <Separator className="my-5" />}
              </div>
            </div>
          ))}
        </SidebarNavInner>
      </ScrollArea>
    </SidebarNavOuter>
  );
}

function SidebarNavOuter({ children }: { children: React.ReactNode }) {
  return (
    <aside className="transition-spacing sticky top-0 z-30 h-[calc(100vh-3.6rem)] w-full shrink-0 duration-500 md:block md:gap-y-2">
      {children}
    </aside>
  );
}

type SidebarNavInnerProps = React.HTMLAttributes<HTMLElement> & {
  children: React.ReactNode;
  className?: string;
};

function SidebarNavInner({ children, className, ...props }: SidebarNavInnerProps) {
  return (
    <nav className={cn("lg:flex-col", className)} {...props}>
      {children}
    </nav>
  );
}
