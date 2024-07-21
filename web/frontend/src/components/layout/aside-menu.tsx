/**
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

// TODO(woflred): refactor this, this shit is cursed.

import { useUserPermissions } from "@/context/user-permissions";
import {
  adminNavLinks,
  billingNavLinks,
  dispatchNavLinks,
  equipmentNavLinks,
  shipmentNavLinks,
} from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { useHeaderStore } from "@/stores/HeaderStore";
import { MenuData, NavLinkGroup, type LinkData } from "@/types/sidebar-nav";
import { faGrid2, faMinus, faPlus } from "@fortawesome/pro-regular-svg-icons";
import { ChevronLeft, ChevronRight, MenuIcon } from "lucide-react";
import {
  createContext,
  memo,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Icon } from "../common/icons";
import { Button } from "../ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { ScrollArea } from "../ui/scroll-area";
import { Sheet, SheetContent, SheetTrigger } from "../ui/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";
import { MiniOrganizationLogo, OrganizationLogo } from "./logo";
import { SearchButton, SiteSearchInput } from "./site-search";
import { UserAsideMenu } from "./user-aside-menu";
import { WorkflowPlaceholder } from "./workspace";

const MenuContext = createContext({
  isMinimized: false,
  toggleMinimize: () => {},
});

const SubMenu = memo(
  ({ links, onLinkClick }: { links: LinkData[]; onLinkClick: () => void }) => {
    const navigate = useNavigate();
    const location = useLocation();

    const handleItemClick = (link: string) => {
      onLinkClick();
      navigate(link);
    };

    return (
      <ul className="border-border relative ml-2 space-y-1 border-l border-dashed pl-2 transition-all duration-300 ease-in-out">
        {links.map((subLink) => {
          const isActive = location.pathname === subLink.link;
          return (
            <li
              key={subLink.key}
              className={cn(
                "relative flex items-center text-muted-foreground cursor-pointer rounded-lg select-none p-2 text-sm transition-all",
                isActive
                  ? "bg-muted/10 text-primary border border-border"
                  : "hover:bg-muted/10 hover:text-primary border border-transparent hover:border-border",
              )}
              onClick={() => handleItemClick(subLink.link || "#")}
            >
              {isActive && (
                <div className="bg-primary absolute left-[-14px] top-1/2 size-2 -translate-y-1/2 rounded-full" />
              )}
              {subLink.label}
            </li>
          );
        })}
      </ul>
    );
  },
);
const MenuItem = memo(
  ({
    item,
    onLinkClick,
    isActive,
  }: {
    item: MenuData;
    onLinkClick: () => void;
    isActive: boolean;
  }) => {
    const { isMinimized } = useContext(MenuContext);

    if (item.content) {
      return (
        <div>
          {!isMinimized && (
            <h3 className="text-muted-foreground select-none text-sm font-semibold uppercase">
              {item.label}
            </h3>
          )}
          {item.content}
        </div>
      );
    }

    if (item.link && item.onClick) {
      throw new Error(
        "MenuItem cannot have both link and onClick. Only one is allowed.",
      );
    }

    return (
      <li
        className={cn(
          "group select-none list-none rounded-md p-2 hover:cursor-pointer",
          isActive
            ? "bg-muted/10 text-primary border border-border"
            : "hover:bg-muted/10 hover:text-primary border border-transparent hover:border-border",
        )}
      >
        {item.link ? (
          <Link
            to={item.link}
            onClick={onLinkClick}
            className={cn(
              "flex w-full items-center leading-6",
              isActive
                ? "text-primary"
                : "text-muted-foreground/80 group-hover:text-primary",
            )}
          >
            {item.icon}
            {!isMinimized && <span className="ml-2 text-sm">{item.label}</span>}
          </Link>
        ) : (
          <button
            onClick={item.onClick}
            className={cn(
              "flex w-full items-center leading-6",
              isActive
                ? "text-primary"
                : "text-muted-foreground/80 group-hover:text-primary",
            )}
          >
            {item.icon}
            {!isMinimized && <span className="ml-2 text-sm">{item.label}</span>}
          </button>
        )}
      </li>
    );
  },
);

const LinksComponent = ({
  linkData,
  onLinkClick,
}: {
  linkData: NavLinkGroup[];
  onLinkClick: () => void;
}) => {
  const [openSubMenu, setOpenSubMenu] = useState<string | null>(null);
  const location = useLocation();
  const { isMinimized } = useContext(MenuContext);
  const { userHasPermission } = useUserPermissions();

  const handleToggleSubMenu = useCallback(
    (label: string) => {
      setOpenSubMenu(openSubMenu === label ? null : label);
    },
    [openSubMenu],
  );

  useEffect(() => {
    let found = false;
    linkData.forEach((mainItem) => {
      mainItem.links.forEach((linkItem) => {
        if (linkItem.subLinks) {
          linkItem.subLinks.forEach((subLink) => {
            if (location.pathname === subLink.link) {
              setOpenSubMenu(linkItem.menuKey);
              found = true;
            }
          });
        } else if (location.pathname === linkItem.link) {
          found = true;
        }
      });
    });
    if (!found) {
      setOpenSubMenu(null);
    }
  }, [location.pathname, linkData]);

  const renderLink = useCallback(
    (linkItem: LinkData) => {
      if (linkItem.permission && !userHasPermission(linkItem.permission)) {
        return null;
      }

      if (!linkItem.subLinks) {
        return (
          <MenuItem
            key={linkItem.menuKey}
            item={linkItem}
            onLinkClick={onLinkClick}
            isActive={location.pathname === linkItem.link}
          />
        );
      }

      return (
        <li key={linkItem.menuKey} className="space-y-1">
          {isMinimized ? (
            <DropdownMenu>
              <DropdownMenuTrigger className="hover:bg-muted focus:bg-muted flex items-center rounded-lg p-2 hover:cursor-pointer">
                {linkItem.icon}
              </DropdownMenuTrigger>
              <DropdownMenuContent side="right">
                {linkItem.subLinks.map((subLink) => (
                  <DropdownMenuItem
                    key={subLink.key}
                    onSelect={() => onLinkClick()}
                  >
                    <Link
                      to={subLink.link}
                      className="flex w-full items-center text-sm leading-6"
                    >
                      {subLink.label}
                    </Link>
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <>
              <div
                onClick={() => handleToggleSubMenu(linkItem.menuKey)}
                className={cn(
                  "text-muted-foreground/80 hover:text-primary flex cursor-pointer select-none items-center justify-between rounded-md p-2 leading-6",
                  openSubMenu === linkItem.menuKey ? "text-primary" : "",
                )}
              >
                <div className="flex">
                  <div className="pr-2">{linkItem.icon}</div>
                  <span className="text-sm">{linkItem.label}</span>
                </div>
                {openSubMenu === linkItem.menuKey ? (
                  <Icon
                    icon={faMinus}
                    className={cn(
                      "size-3 transition-transform duration-300 ease-in-out",
                    )}
                  />
                ) : (
                  <Icon
                    icon={faPlus}
                    className={cn(
                      "size-3 transition-transform duration-300 ease-in-out",
                    )}
                  />
                )}
              </div>
              {linkItem.subLinks && (
                <div
                  className={cn(
                    "overflow-hidden transition-all duration-300 ease-in-out",
                    openSubMenu === linkItem.menuKey
                      ? "max-h-screen"
                      : "max-h-0",
                  )}
                >
                  <SubMenu
                    links={linkItem.subLinks}
                    onLinkClick={onLinkClick}
                  />
                </div>
              )}
            </>
          )}
        </li>
      );
    },
    [
      onLinkClick,
      userHasPermission,
      openSubMenu,
      handleToggleSubMenu,
      location.pathname,
      isMinimized,
    ],
  );

  const permittedLinks = useMemo(
    () =>
      linkData.flatMap((mainItem) =>
        mainItem.links.map(renderLink).filter(Boolean),
      ),
    [linkData, renderLink],
  );

  return <ul className="space-y-1">{permittedLinks}</ul>;
};

function Menu({
  menuItems,
  onLinkClick,
}: {
  menuItems: MenuData[];
  onLinkClick: () => void;
}) {
  const location = useLocation();

  return (
    <div className="bg-transparent sm:rounded-md">
      <ul className="space-y-6">
        {menuItems.map((item) => (
          <MenuItem
            key={item.menuKey}
            item={item}
            onLinkClick={onLinkClick}
            isActive={location.pathname === item.link}
          />
        ))}
      </ul>
    </div>
  );
}

export function AsideMenu() {
  const [open, setMenuOpen] = useHeaderStore.use("asideMenuOpen");
  // const location = useLocation();

  const toggleMenu = () => {
    setMenuOpen(!open);
  };

  const menuItems: MenuData[] = [
    {
      menuKey: "dashboardMenu",
      label: "Dashboard",
      link: "/",
      icon: <Icon icon={faGrid2} />,
    },
    {
      menuKey: "billingMenu",
      label: "Billing & AR",
      content: (
        <LinksComponent linkData={billingNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "dispatchMenu",
      label: "Dispatch Management",
      content: (
        <LinksComponent linkData={dispatchNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "equipmentMenu",
      label: "Equipment Management",
      content: (
        <LinksComponent linkData={equipmentNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "shipmentMenu",
      label: "Shipment Management",
      content: (
        <LinksComponent linkData={shipmentNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "safetyMenu",
      label: "Safety OS&D",
      content: (
        <LinksComponent linkData={shipmentNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "reportingMenu",
      label: "Reporting & Analytics",
      content: (
        <LinksComponent linkData={shipmentNavLinks} onLinkClick={toggleMenu} />
      ),
    },
    {
      menuKey: "adminMenu",
      label: "Administration",
      content: (
        <LinksComponent linkData={adminNavLinks} onLinkClick={toggleMenu} />
      ),
    },
  ];

  return (
    <div className="relative flex grow flex-col overflow-hidden">
      <ScrollArea className="relative size-full grow px-4">
        <div className="pb-10">
          <Menu menuItems={menuItems} onLinkClick={toggleMenu} />
        </div>

        <div className="from-background pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t to-transparent" />
      </ScrollArea>
      {/* <MenuItem item={logoutItem} onLinkClick={toggleMenu} /> */}
      <UserAsideMenu />
    </div>
  );
}

function AsideMenuButton() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Button
              size="icon"
              variant="outline"
              className="border-muted-foreground/40 hover:border-muted-foreground/80 group relative size-8"
            >
              <MenuIcon className="text-muted-foreground group-hover:text-foreground size-5" />
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Menu</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function AsideMenuDialog() {
  return (
    <Sheet>
      <SheetTrigger className="flex xl:hidden">
        <AsideMenuButton />
      </SheetTrigger>
      <SheetContent side="left" className="w-[20em] overflow-y-scroll">
        <aside className="overflow-auto">
          <div className="border-border mb-4 border-b border-dashed pb-4">
            <OrganizationLogo />
          </div>
          <AsideMenu />
        </aside>
      </SheetContent>
    </Sheet>
  );
}

function MainAsideMenu() {
  const [isMinimized, setIsMinimized] = useState(false);

  const toggleMinimize = () => {
    setIsMinimized(!isMinimized);
  };

  return (
    <MenuContext.Provider value={{ isMinimized, toggleMinimize }}>
      <div
        className={cn(
          "bg-background border-border hidden h-screen shrink-0 flex-col border-r xl:flex transition-all duration-300 ease-in-out",
          isMinimized ? "w-[70px]" : "w-72",
        )}
      >
        <div className="border-border relative mb-4 border-b border-dashed p-4">
          {isMinimized ? <MiniOrganizationLogo /> : <OrganizationLogo />}
          <span className="text-muted-foreground mt-4 block text-xs">
            {isMinimized ? <SearchButton /> : <SiteSearchInput />}
          </span>
          <Button
            className="absolute -right-4 top-10 size-8 rounded-full p-1"
            onClick={toggleMinimize}
            size="icon"
            variant="outline"
          >
            {isMinimized ? (
              <ChevronRight className="text-muted-foreground size-4" />
            ) : (
              <ChevronLeft className="text-muted-foreground size-4" />
            )}
          </Button>
        </div>
        <div className={cn("p-4", isMinimized && "hidden")}>
          <WorkflowPlaceholder />
        </div>
        <AsideMenu />
      </div>
    </MenuContext.Provider>
  );
}

export default MainAsideMenu;
