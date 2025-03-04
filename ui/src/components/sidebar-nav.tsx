import { useDebouncedCallback } from "@/hooks/use-debounce";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import React, { useEffect, useMemo, useState } from "react";
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
  const [isScrolled, setIsScrolled] = useState<boolean>(false);

  const debouncedHandleScroll = useDebouncedCallback(() => {
    if (window.scrollY > 80) {
      setIsScrolled(true);
    } else {
      setIsScrolled(false);
    }
  }, 100);

  useEffect(() => {
    const handleScroll = () => debouncedHandleScroll.setValue();
    window.addEventListener("scroll", handleScroll);

    return () => {
      window.removeEventListener("scroll", handleScroll);
      debouncedHandleScroll.cancel();
    };
  }, [debouncedHandleScroll]);

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
    <aside
      className={cn(
        "transition-spacing fixed z-30 -ml-2 hidden w-full shrink-0 duration-500 md:sticky md:block",
        isScrolled ? "pt-40 h-[calc(100vh+100px)]" : "h-[calc(100vh-5rem)]",
      )}
    >
      <ScrollArea className="bg-card text-card-foreground size-full rounded-lg border p-3">
        <nav className={cn("lg:flex-col", className)} {...props}>
          {Object.entries(groupedLinks).map(
            ([group, groupLinks], index, array) => (
              <div key={group} className="space-y-2">
                {group !== "ungrouped" && (
                  <h3 className="text-muted-foreground select-none text-sm font-semibold uppercase">
                    {group}
                  </h3>
                )}
                <div>
                  {groupLinks.map((link) => (
                    <Link
                      key={link.title}
                      to={link.href}
                      className={cn(
                        buttonVariants({ variant: "ghost" }),
                        location.pathname === link.href
                          ? "bg-accent"
                          : "hover:bg-accent",
                        link.disabled && "cursor-not-allowed opacity-50",
                        "group justify-start flex items-center text-sm mb-1",
                      )}
                    >
                      {link.title}
                    </Link>
                  ))}
                  {index !== array.length - 1 && <Separator className="my-5" />}
                </div>
              </div>
            ),
          )}
        </nav>
      </ScrollArea>
    </aside>
  );
}
