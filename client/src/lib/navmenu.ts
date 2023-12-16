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

import React from "react";

export type Permission = string | undefined;

export interface SubLink {
  label: string;
  link: string;
  permission?: Permission;
  description: string;
}

export interface MainLink extends Omit<SubLink, "link" | "description"> {
  link?: string;
  subLinks?: SubLink[];
}

export type MainItem = {
  links: MainLink[];
};

export type MenuContent = React.ReactNode | MainItem[];

export type MenuData = {
  menuKey: string;
  label: string;
  permission?: Permission;
  content?: React.ReactNode;
  link?: string;
  footerContent?: React.ReactNode;
};

export type NavigationMenuItemProps = {
  data: MenuData;
  setMenuOpen: React.Dispatch<React.SetStateAction<string | undefined>>;
  setMenuPosition: (position: { left: number; width: number }) => void;
  ref: React.Ref<HTMLLIElement>;
  menuItemRefs: React.MutableRefObject<Record<string, HTMLLIElement>>;
};

export const hasPermission = (
  item: { permission?: Permission },
  userHasPermission: (permission: string) => boolean,
): boolean => !item.permission || userHasPermission(item.permission);

export function userHasAccessToContent(
  content: MenuContent,
  userHasPermission: (permission: string) => boolean,
  isAdmin: boolean,
) {
  if (isAdmin) return true;

  if (Array.isArray(content)) {
    return content.some((mainItem) =>
      mainItem.links.some(
        (link) =>
          hasPermission(link, userHasPermission) &&
          (!link.subLinks ||
            link.subLinks.some((subLink) =>
              hasPermission(subLink, userHasPermission),
            )),
      ),
    );
  }
  return true;
}

export function calculatePosition(element: HTMLLIElement) {
  const rect = element.getBoundingClientRect();
  const parentRect = element.offsetParent?.getBoundingClientRect();
  const leftPosition = parentRect ? rect.left - parentRect.left : rect.left;
  return {
    left: leftPosition,
    width: element.offsetWidth,
  };
}
