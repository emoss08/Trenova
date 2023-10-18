/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import { ChevronsLeftIcon } from "lucide-react";
import React, { useState } from "react";
import { ListItem } from "./links-group";
import { useHeaderStore } from "@/stores/HeaderStore";
// Type Definitions

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

/**
 * A ProtectedLink component which checks for permissions before rendering.
 */
export const ProtectedLink: React.FC<LinkData & { onClick?: () => void }> = ({
  label,
  link,
  description,
  permission,
  onClick,
}) => {
  const { userHasPermission } = useUserPermissions();

  if (permission && !userHasPermission(permission)) {
    return null;
  }

  return (
    <ListItem title={label} to={link} onClick={onClick}>
      {description}
    </ListItem>
  );
};

/**
 * A SingleLink component which renders an individual link.
 */
const SingleLink: React.FC<{
  subItem: LinkData;
  setActiveSubLinks: (links: LinkData[] | null) => void;
}> = ({ subItem, setActiveSubLinks }) => (
  <ProtectedLink
    {...subItem}
    onClick={() => subItem.subLinks && setActiveSubLinks(subItem.subLinks)}
  />
);

/**
 * The LinksComponent renders a list of links.
 */
export function LinksComponent({ linkData }: LinksComponentProps) {
  const [activeSubLinks, setActiveSubLinks] = useState<LinkData[] | null>(null);
  const { userHasPermission } = useUserPermissions();
  // Checks if user has permission to any of the subLinks
  const userHasSubLinkPermission = (subLinks: LinkData[]): boolean =>
    subLinks.some(
      (subLink) => !subLink.permission || userHasPermission(subLink.permission),
    );

  // Handler for the back click, to navigate back from sublinks
  const handleBackClick = () => setActiveSubLinks(null);

  // Filter and map link data to permitted links
  const permittedLinks = linkData.flatMap((mainItem) =>
    mainItem.links.flatMap((subItem) => {
      if (subItem.permission && !userHasPermission(subItem.permission)) {
        return [];
      }
      if (subItem.subLinks && !userHasSubLinkPermission(subItem.subLinks)) {
        return [];
      }
      return (
        <SingleLink
          key={subItem.label}
          subItem={subItem}
          setActiveSubLinks={setActiveSubLinks}
        />
      );
    }),
  );

  return (
    <ul
      className={`relative grid w-[400px] gap-3 p-4 ${
        activeSubLinks ? "pt-8" : ""
      } md:w-[500px] md:grid-cols-2 lg:w-[600px]`}
    >
      {!activeSubLinks ? (
        permittedLinks
      ) : (
        <>
          <button
            onClick={handleBackClick}
            className="absolute top-2 right-2 rounded-md text-sm transition duration-200 z-10"
          >
            <ChevronsLeftIcon className="w-5 h-5" />
          </button>
          {activeSubLinks.map((subLink) => (
            <li key={subLink.label}>
              <ListItem
                onClick={() => useHeaderStore.set("menuOpen", undefined)}
                title={subLink.label}
                to={subLink.link}
                permission={subLink.permission}
              >
                {subLink.description}
              </ListItem>
            </li>
          ))}
        </>
      )}
    </ul>
  );
}
