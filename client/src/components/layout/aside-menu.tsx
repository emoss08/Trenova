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
import { OrganizationNameLogo } from "@/components/layout/logo";
import TeamSwitcher from "@/components/layout/team-switcher";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useUserPermissions } from "@/context/user-permissions";
import {
  billingNavLinks,
  dispatchNavLinks,
  equipmentNavLinks,
  shipmentNavLinks,
} from "@/lib/nav-links";
import { useHeaderStore } from "@/stores/HeaderStore";
import { faGrid2, faUserCrown } from "@fortawesome/pro-duotone-svg-icons";
import { faBars } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { ChevronDownIcon } from "lucide-react";
import React, { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../ui/button";

type SubLinkData = {
  key: string;
  label: string;
  link: string;
  permission?: string;
  description?: string;
};

export type LinkData = {
  key: string;
  label: string;
  link: string;
  permission?: string;
  description?: string;
  icon?: React.ReactNode;
  subLinks?: SubLinkData[];
};

export type LinksComponentProps = {
  linkData: {
    links: LinkData[];
  }[];
};

type MenuData = {
  menuKey: string;
  label: string;
  link?: string;
  content?: React.ReactNode;
  permission?: string;
  icon?: React.ReactNode;
};

const SubMenu = React.memo(
  ({ links, onLinkClick }: { links: LinkData[]; onLinkClick: () => void }) => (
    <ul className="pl-4">
      {links.map((subLink) => (
        <li
          key={subLink.key}
          className="select-none rounded-md p-2 hover:bg-muted focus:bg-muted"
        >
          <Link
            to={subLink.link || "#"}
            onClick={onLinkClick}
            className="block w-full text-sm leading-6"
          >
            {subLink.label}
          </Link>
        </li>
      ))}
    </ul>
  ),
);

const MenuItem = React.memo(
  ({ item, onLinkClick }: { item: MenuData; onLinkClick: () => void }) => {
    if (item.content) {
      return (
        <div>
          <h3 className="p-2 font-semibold">{item.label}</h3>
          {item.content}
        </div>
      );
    }

    return (
      <li className="select-none rounded-md p-2 hover:bg-accent focus:bg-accent">
        <Link
          to={item.link || "#"}
          onClick={onLinkClick}
          className="flex w-full items-center text-sm leading-6"
        >
          {item.icon}
          <span className="ml-2">{item.label}</span>
        </Link>
      </li>
    );
  },
);

const LinksComponent = ({
  linkData,
  onLinkClick,
}: LinksComponentProps & { onLinkClick: () => void }) => {
  const [openSubMenu, setOpenSubMenu] = useState<string | null>(null);

  const handleToggleSubMenu = React.useCallback(
    (label: string) => {
      setOpenSubMenu(openSubMenu === label ? null : label);
    },
    [openSubMenu],
  );

  const { userHasPermission } = useUserPermissions();

  const renderLink = React.useCallback(
    (linkItem: LinkData) => {
      if (linkItem.permission && !userHasPermission(linkItem.permission)) {
        return null;
      }

      if (!linkItem.subLinks) {
        return (
          <li
            key={linkItem.key}
            className="select-none rounded-md p-2 hover:bg-accent focus:bg-accent"
          >
            <Link
              to={linkItem.link || "#"}
              onClick={onLinkClick}
              className="flex w-full items-center text-sm leading-6"
            >
              {linkItem.icon}
              <span className="ml-2">{linkItem.label}</span>
            </Link>
          </li>
        );
      }

      return (
        <>
          <li key={linkItem.key} className="space-y-2">
            <div
              onClick={() => handleToggleSubMenu(linkItem.key)}
              className="flex cursor-pointer select-none items-center justify-between rounded-md p-2 text-sm leading-6 hover:bg-accent focus:bg-accent"
            >
              <div className="flex">
                <div className="pr-2">{linkItem.icon}</div>
                {linkItem.label}
              </div>
              <ChevronDownIcon
                className={`size-4 ${
                  openSubMenu === linkItem.key ? "rotate-180" : ""
                }`}
              />
            </div>
            {linkItem.subLinks && openSubMenu === linkItem.key && (
              <SubMenu links={linkItem.subLinks} onLinkClick={onLinkClick} />
            )}
          </li>
          <Separator className="my-2" />
        </>
      );
    },
    [onLinkClick, userHasPermission, openSubMenu, handleToggleSubMenu],
  );

  const permittedLinks = React.useMemo(
    () =>
      linkData.flatMap((mainItem) =>
        mainItem.links.map(renderLink).filter(Boolean),
      ),
    [linkData, renderLink],
  );

  return <ul className="space-y-1">{permittedLinks}</ul>;
};

function AsideMenu({
  menuItems,
  onLinkClick,
}: {
  menuItems: MenuData[];
  onLinkClick: () => void;
}) {
  return (
    <div className="mt-5 overflow-hidden bg-background sm:rounded-md">
      <ul>
        {menuItems.map((item) => (
          <MenuItem key={item.menuKey} item={item} onLinkClick={onLinkClick} />
        ))}
      </ul>
    </div>
  );
}

export function AsideMenuSheet() {
  const [open, setMenuOpen] = useHeaderStore.use("asideMenuOpen");

  const toggleMenu = () => {
    setMenuOpen(!open);
  };

  const menuItems: MenuData[] = [
    {
      menuKey: "dashboardMenu",
      label: "Dashboard",
      link: "/",
      icon: <FontAwesomeIcon icon={faGrid2} className="size-4" />,
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
      menuKey: "adminMenu",
      label: "Administrator",
      link: "/admin/dashboard/",
      permission: "view_admin_dashboard",
      icon: <FontAwesomeIcon icon={faUserCrown} className="size-4" />,
    },
  ];

  return (
    <Sheet open={open} onOpenChange={setMenuOpen}>
      <SheetTrigger asChild>
        <Button
          size="icon"
          variant="outline"
          className="flex size-9 border-muted-foreground/40 hover:border-muted-foreground/80 md:hidden"
        >
          <FontAwesomeIcon icon={faBars} className="size-5" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[400px] overflow-auto" side="left">
        <SheetHeader>
          <SheetTitle>
            <OrganizationNameLogo />
            <div className="mt-2 border-b border-muted-foreground/40" />
            <TeamSwitcher className="mt-2" />
          </SheetTitle>
        </SheetHeader>
        <AsideMenu menuItems={menuItems} onLinkClick={toggleMenu} />
      </SheetContent>
    </Sheet>
  );
}
