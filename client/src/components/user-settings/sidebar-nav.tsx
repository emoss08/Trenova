import { cn } from "@/lib/utils";
import { type SidebarLink } from "@/types/sidebar-nav";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { debounce } from "lodash-es";
import React, { useEffect } from "react";
import { Link, useLocation } from "react-router-dom";
import { buttonVariants } from "../ui/button";
import { ScrollArea } from "../ui/scroll-area";

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
  }, []);

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
                      link.disabled && "cursor-not-allowed opacity-50",
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

type LinkGroup = {
  key: string;
  href: string;
  title: string;
  component: React.ReactNode;
  icon: any;
};

type LinkGroupProps = {
  title: string;
  links: LinkGroup[];
};

export function ModalAsideMenu({
  heading,
  linkGroups,
  activeTab,
  setActiveTab,
}: {
  heading: string;
  linkGroups: LinkGroupProps[];
  activeTab: string;
  setActiveTab: (tabId: string) => void;
}) {
  return (
    <div className="transition-spacing fixed top-14 z-30 -ml-2 hidden size-full h-[600px] shrink-0 duration-500 md:sticky md:block">
      <ScrollArea className="bg-card text-card-foreground size-full border-r">
        <h2 className="pb-4 text-base">{heading}</h2>
        {linkGroups.map((group) => (
          <div key={group.title}>
            <h3 className="text-muted-foreground px-2 py-1 text-xs font-semibold uppercase">
              {group.title}
            </h3>
            <nav className="space-y-1">
              {group.links.map((link) => (
                <Link
                  key={link.key}
                  to={link.href}
                  className={cn(
                    buttonVariants({ variant: "ghost" }),
                    "group justify-start flex items-center space-x-2 mr-2 p-2 rounded-lg transition-colors",
                    "hover:bg-muted",
                    { "bg-muted text-primary": activeTab === link.key },
                  )}
                  onClick={() => setActiveTab(link.key)}
                >
                  <FontAwesomeIcon icon={link.icon} className="size-4" />
                  <span>{link.title}</span>
                </Link>
              ))}
            </nav>
          </div>
        ))}
      </ScrollArea>
    </div>
  );
}
