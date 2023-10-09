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
import React, { memo, useState } from "react";
import { ListItem } from "./links-group";

export type LinkData = {
  label: string;
  link: string;
  permission?: string;
  description?: string;
  subLinks?: LinkData[];
};

export type LinksComponentProps = {
  linkData: {
    links: LinkData[];
  }[];
};

// Centralized permission check for links
export const ProtectedLink: React.FC<LinkData & { onClick?: () => void }> = (
  props,
) => {
  const { userHasPermission } = useUserPermissions();

  if (props.permission && !userHasPermission(props.permission)) {
    return null;
  }

  return (
    <ListItem title={props.label} href={props.link} onClick={props.onClick}>
      {props.description}
    </ListItem>
  );
};
const SingleLink = memo(({ subItem, setActiveSubLinks }: any) => {
  return (
    <ProtectedLink
      {...subItem}
      onClick={() => {
        if (subItem.subLinks) {
          setActiveSubLinks(subItem.subLinks);
        }
      }}
    />
  );
});

export function LinksComponent({ linkData }: LinksComponentProps) {
  const [activeSubLinks, setActiveSubLinks] = useState<Array<LinkData> | null>(
    null,
  );
  const { userHasPermission } = useUserPermissions();

  const userHasSubLinkPermission = (subLinks: LinkData[]): boolean => {
    return subLinks.some(
      (subLink) => !subLink.permission || userHasPermission(subLink.permission),
    );
  };

  const handleBackClick = () => {
    setActiveSubLinks(null);
  };

  const permittedLinks = linkData.flatMap((mainItem, mainIndex) =>
    mainItem.links.flatMap((subItem, subIndex) => {
      if (subItem.permission && !userHasPermission(subItem.permission)) {
        return [];
      }

      if (subItem.subLinks && !userHasSubLinkPermission(subItem.subLinks)) {
        return [];
      }

      return (
        <SingleLink
          key={`link-${mainIndex}-${subIndex}`}
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
          {activeSubLinks.map((subLinkItem) => (
            <li key={subLinkItem.label}>
              <ListItem
                title={subLinkItem.label}
                href={subLinkItem.link}
                permission={subLinkItem.permission}
              >
                {subLinkItem.description}
              </ListItem>
            </li>
          ))}
        </>
      )}
    </ul>
  );
}
