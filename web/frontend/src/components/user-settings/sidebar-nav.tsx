import { cn } from "@/lib/utils";
import { LinkGroupProps, type SidebarLink } from "@/types/sidebar-nav";
import { debounce } from "lodash-es";
import React, { useEffect } from "react";
import { Link, useLocation } from "react-router-dom";
import { buttonVariants } from "../ui/button";
import { ScrollArea } from "../ui/scroll-area";
import { Separator } from "../ui/separator";

interface SidebarNavProps extends React.HTMLAttributes<HTMLElement> {
  links: SidebarLink[];
}

export function SidebarNav({ className, links, ...props }: SidebarNavProps) {
  const location = useLocation();
  const [isScrolled, setIsScrolled] = React.useState(false);

  const debouncedHandleScroll = debounce(() => {
    if (window.scrollY > 80) {
      setIsScrolled(true);
    } else {
      setIsScrolled(false);
    }
  }, 100);

  useEffect(() => {
    const handleScroll = () => debouncedHandleScroll();
    window.addEventListener("scroll", handleScroll);

    return () => {
      window.removeEventListener("scroll", handleScroll);
      debouncedHandleScroll.cancel && debouncedHandleScroll.cancel();
    };
  }, [debouncedHandleScroll]);

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
        <nav className={cn("lg:flex-col", className)} {...props}>
          {Object.entries(groupedLinks).map(
            ([group, groupLinks], index, array) => (
              <div key={group} className="space-y-2">
                {group !== "ungrouped" && (
                  <h3 className="text-muted-foreground ml-4 select-none text-sm font-semibold uppercase">
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
                          ? "bg-muted"
                          : "hover:bg-muted",
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

export function ModalAsideMenu({
  linkGroups,
  activeTab,
  setActiveTab,
}: {
  linkGroups: LinkGroupProps[];
  activeTab: string;
  setActiveTab: (tabId: string) => void;
}) {
  return (
    <nav className="transition-spacing fixed top-14 z-30 -ml-2 hidden size-full h-[600px] shrink-0 duration-500 md:sticky md:block">
      <ScrollArea className="bg-card text-card-foreground size-full border-r">
        {linkGroups.map((group, index, array) => (
          <div key={group.title}>
            <h3 className="text-muted-foreground select-none text-sm font-semibold uppercase">
              {group.title}
            </h3>
            {group.links.map((link) => (
              <Link
                key={link.key}
                to={link.href}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "sm" }),
                  "hover:bg-muted",
                  { "bg-muted text-primary": activeTab === link.key },
                  "group justify-start flex items-center text-sm mb-1 p-2 mr-4",
                )}
                onClick={() => setActiveTab(link.key)}
              >
                <span>{link.title}</span>
              </Link>
            ))}
            {index !== array.length - 1 && <Separator className="my-2" />}
          </div>
        ))}
      </ScrollArea>
    </nav>
  );
}
