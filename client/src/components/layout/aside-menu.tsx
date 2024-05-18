import { useUserPermissions } from "@/context/user-permissions";
import {
  billingNavLinks,
  dispatchNavLinks,
  equipmentNavLinks,
  shipmentNavLinks,
} from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { useUserStore } from "@/stores/AuthStore";
import { useHeaderStore } from "@/stores/HeaderStore";
import { faChevronDown, faGrid2 } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { memo, useCallback, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { OrganizationLogo } from "./logo";
import { SiteSearchInput } from "./site-search";
import { UserAvatarMenu } from "./user-avatar-menu";

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
const SubMenu = memo(
  ({ links, onLinkClick }: { links: LinkData[]; onLinkClick: () => void }) => {
    const navigate = useNavigate();

    const handleItemClick = (link: string) => {
      onLinkClick();
      navigate(link);
    };

    return (
      <ul className="space-y-1 pl-10">
        {links.map((subLink) => (
          <li
            key={subLink.key}
            className="text-muted-foreground hover:text-primary cursor-pointer select-none list-disc rounded-md p-2 text-sm"
            onClick={() => handleItemClick(subLink.link || "#")}
          >
            {subLink.label}
          </li>
        ))}
      </ul>
    );
  },
);

const MenuItem = memo(
  ({ item, onLinkClick }: { item: MenuData; onLinkClick: () => void }) => {
    if (item.content) {
      return (
        <div>
          <h3 className="text-muted-foreground select-none text-sm font-semibold uppercase">
            {item.label}
          </h3>
          {item.content}
        </div>
      );
    }

    return (
      <li className="hover:bg-accent focus:bg-accent select-none rounded-md p-2">
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

  const handleToggleSubMenu = useCallback(
    (label: string) => {
      setOpenSubMenu(openSubMenu === label ? null : label);
    },
    [openSubMenu],
  );

  const { userHasPermission } = useUserPermissions();

  const renderLink = useCallback(
    (linkItem: LinkData) => {
      if (linkItem.permission && !userHasPermission(linkItem.permission)) {
        return null;
      }

      if (!linkItem.subLinks) {
        return (
          <li
            key={linkItem.key}
            className="hover:bg-accent focus:bg-accent select-none rounded-md p-2"
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
              className="hover:bg-accent focus:bg-accent flex cursor-pointer select-none items-center justify-between rounded-md p-2 text-sm leading-6"
            >
              <div className="flex">
                <div className="pr-2">{linkItem.icon}</div>
                {linkItem.label}
              </div>
              <FontAwesomeIcon
                icon={faChevronDown}
                className={cn(
                  "size-2",
                  openSubMenu === linkItem.key ? "rotate-180" : "",
                )}
              />
            </div>
            {linkItem.subLinks && openSubMenu === linkItem.key && (
              <SubMenu links={linkItem.subLinks} onLinkClick={onLinkClick} />
            )}
          </li>
        </>
      );
    },
    [onLinkClick, userHasPermission, openSubMenu, handleToggleSubMenu],
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
  return (
    <div className="mt-5 bg-transparent sm:rounded-md">
      <ul className="space-y-10">
        {menuItems.map((item) => (
          <MenuItem key={item.menuKey} item={item} onLinkClick={onLinkClick} />
        ))}
      </ul>
    </div>
  );
}

export function AsideMenu() {
  const [open, setMenuOpen] = useHeaderStore.use("asideMenuOpen");

  const toggleMenu = () => {
    setMenuOpen(!open);
  };

  const [user] = useUserStore.use("user");

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
  ];

  // const adminMenuItem: MenuData = {
  //   menuKey: "adminMenu",
  //   label: "Administrator",
  //   link: "/admin/dashboard/",
  //   permission: "view_admin_dashboard",
  //   icon: <FontAwesomeIcon icon={faUserCrown} className="size-4" />,
  // };

  return (
    <aside
      className="bg-background h-screen w-[300px] overflow-auto border-r border-dashed p-4"
      aria-label="Sidebar"
    >
      <div className="border-border mb-4 border-b border-dashed pb-4">
        <OrganizationLogo />
      </div>
      <div className="border-border flex flex-col border-b pb-4">
        <UserAvatarMenu user={user} />
      </div>
      <div className="mt-4 flex justify-center">
        <SiteSearchInput />
      </div>
      <Menu menuItems={menuItems} onLinkClick={toggleMenu} />
    </aside>
  );
}
