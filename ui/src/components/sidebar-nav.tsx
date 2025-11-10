import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import React, { useMemo } from "react";
import { Link, useLocation } from "react-router";
import { ScrollArea } from "./ui/scroll-area";
import { Separator } from "./ui/separator";

export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
};

type SidebarNavProps = React.HTMLAttributes<HTMLElement> & {
  links: SidebarLink[];
};

export function SidebarNav({ links, className, ...props }: SidebarNavProps) {
  const location = useLocation();

  const groupedLinks = useMemo(() => {
    type GroupedLinks = Record<string, SidebarLink[]>;

    return links.reduce((acc: GroupedLinks, link) => {
      const groupName = link.group || "ungrouped";
      acc[groupName] = acc[groupName] || [];
      acc[groupName].push(link);
      return acc;
    }, {} as GroupedLinks);
  }, [links]);

  return (
    <SidebarNavOuter>
      <ScrollArea className="h-full rounded-lg border bg-sidebar p-3 text-card-foreground">
        <SidebarNavInner className={cn("lg:flex-col", className)} {...props}>
          {Object.entries(groupedLinks).map(
            ([group, groupLinks], index, array) => (
              <div key={group} className="space-y-2">
                {group !== "ungrouped" && (
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase select-none">
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
                          link.disabled && "opacity-50 pointer-events-none",
                          "group justify-start flex items-center text-sm mb-1",
                        )}
                      >
                        {link.title}
                      </Link>
                    );
                  })}
                  {index !== array.length - 1 && <Separator className="my-5" />}
                </div>
              </div>
            ),
          )}
        </SidebarNavInner>
      </ScrollArea>
    </SidebarNavOuter>
  );
}

function SidebarNavOuter({ children }: { children: React.ReactNode }) {
  return (
    <aside className="transition-spacing sticky top-0 z-30 -ml-2 h-screen w-full shrink-0 duration-500 md:block md:gap-y-2">
      {children}
    </aside>
  );
}

type SidebarNavInnerProps = React.HTMLAttributes<HTMLElement> & {
  children: React.ReactNode;
  className?: string;
};

function SidebarNavInner({
  children,
  className,
  ...props
}: SidebarNavInnerProps) {
  return (
    <nav className={cn("lg:flex-col", className)} {...props}>
      {children}
    </nav>
  );
}
