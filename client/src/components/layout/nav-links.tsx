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

import { useUserPermissions } from "@/context/user-permissions";
import { cn } from "@/lib/utils";
import { useHeaderStore } from "@/stores/HeaderStore";
import React, { useState } from "react";
import { Button } from "../ui/button";
import { ListItem } from "./links-group";

/**
 * Definition for individual link data.
 */
export type LinkData = {
  label: string;
  link: string;
  permission?: string;
  description?: string;
  subLinks?: LinkData[];
};

/**
 * Props for the LinksComponent.
 */
export type LinksComponentProps = {
  linkData: {
    links: LinkData[];
  }[];
};

export const ProtectedLink: React.FC<
  LinkData & { onClick?: (event: React.MouseEvent) => void }
> = ({ label, link, description, permission, onClick }) => {
  const { userHasPermission } = useUserPermissions();

  if (permission && !userHasPermission(permission)) {
    return null;
  }

  return (
    <ListItem
      title={label}
      to={link}
      onClick={(event) => {
        if (onClick) {
          event.preventDefault();
          onClick(event);
        } else {
          // directly navigate to the link if there is no onClick Handler and sublinks for the menu.
        }
      }}
    >
      {description}
    </ListItem>
  );
};

const SingleLink: React.FC<{
  subItem: LinkData;
  setActiveSubLinks: (links: LinkData[] | null) => void;
}> = ({ subItem, setActiveSubLinks }) => {
  const { userHasPermission } = useUserPermissions();

  const hasAccessibleSubLink = subItem.subLinks?.some(
    (link) => !link.permission || userHasPermission(link.permission),
  );

  if (!hasAccessibleSubLink) return null;

  return (
    <ProtectedLink
      {...subItem}
      onClick={(event) => {
        if (subItem.subLinks) {
          event.preventDefault();
          setActiveSubLinks(subItem.subLinks);
        }
      }}
    />
  );
};

/**
 * The LinksComponent renders a list of links.
 */
export function LinksComponent({ linkData }: LinksComponentProps) {
  const [activeSubLinks, setActiveSubLinks] = useState<LinkData[] | null>(null);
  const { userHasPermission } = useUserPermissions();
  const [, setMenuOpen] = useHeaderStore.use("menuOpen");

  const handleBackClick = () => setActiveSubLinks(null);

  const renderLink = (linkItem: LinkData) => {
    const hasPermission =
      !linkItem.permission || userHasPermission(linkItem.permission);
    const hasAccessibleSubLink = linkItem.subLinks?.some(
      (subLink) => !subLink.permission || userHasPermission(subLink.permission),
    );

    // Do not render the link if it has no permission or if it has sub-links and none are accessible
    if (!hasPermission || (linkItem.subLinks && !hasAccessibleSubLink)) {
      return null;
    }

    return (
      <li key={linkItem.label}>
        {linkItem.subLinks ? (
          <SingleLink
            subItem={linkItem}
            setActiveSubLinks={setActiveSubLinks}
          />
        ) : (
          <ListItem
            title={linkItem.label}
            to={linkItem.link}
            onClick={() => setMenuOpen(undefined)}
          >
            {linkItem.description}
          </ListItem>
        )}
      </li>
    );
  };

  const permittedLinks = linkData.flatMap(
    (mainItem) => mainItem.links.map(renderLink).filter(Boolean), // Filter out null values
  );

  return (
    <ul
      className={cn(
        "relative grid w-[400px] gap-3 p-4",
        activeSubLinks ? "pt-10" : "",
        "md:w-[500px] md:grid-cols-2 lg:w-[700px]",
      )}
    >
      {!activeSubLinks ? (
        permittedLinks
      ) : (
        <>
          <Button
            onClick={handleBackClick}
            className="absolute right-2 top-2 z-10"
            size="xs"
            variant="outline"
          >
            Back
          </Button>
          {activeSubLinks.map((subLink) => (
            <ListItem
              key={subLink.label}
              title={subLink.label}
              to={subLink.link}
              onClick={() => setMenuOpen(undefined)}
            >
              {subLink.description}
            </ListItem>
          ))}
        </>
      )}
    </ul>
  );
}
