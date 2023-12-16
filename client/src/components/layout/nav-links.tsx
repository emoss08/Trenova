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
import { useHeaderStore } from "@/stores/HeaderStore";
import { ChevronsLeftIcon } from "lucide-react";
import React, { useState } from "react";
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
}> = ({ subItem, setActiveSubLinks }) => (
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

/**
 * The LinksComponent renders a list of links.
 */
export function LinksComponent({ linkData }: LinksComponentProps) {
  const [activeSubLinks, setActiveSubLinks] = useState<LinkData[] | null>(null);
  const { userHasPermission } = useUserPermissions();

  // Handler for the back click, to navigate back from sublinks
  const handleBackClick = () => setActiveSubLinks(null);

  // The SingleLink component should only be rendered if permissions allow
  const renderSingleLink = (subItem: LinkData) => {
    if (subItem.permission && !userHasPermission(subItem.permission)) {
      return null;
    }
    return (
      <li key={subItem.label}>
        <SingleLink subItem={subItem} setActiveSubLinks={setActiveSubLinks} />
      </li>
    );
  };

  // Map link data to permitted links, ensuring no empty <li> elements are created
  const permittedLinks = linkData.flatMap(
    (mainItem) => mainItem.links.map(renderSingleLink).filter(Boolean), // Filter out null values
  );

  return (
    <ul
      className={`relative grid w-[400px] gap-3 p-4 ${
        activeSubLinks ? "pt-8" : ""
      } md:w-[500px] md:grid-cols-2 xl:w-[700px]`}
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
          {activeSubLinks.map((subLink) => {
            if (subLink.permission && !userHasPermission(subLink.permission)) {
              return null; // Don't render the list item if permission is not granted
            }
            return (
              <li key={subLink.label}>
                <ListItem
                  onClick={() => useHeaderStore.set("menuOpen", undefined)}
                  title={subLink.label}
                  to={subLink.link}
                >
                  {subLink.description}
                </ListItem>
              </li>
            );
          })}
        </>
      )}
    </ul>
  );
}
