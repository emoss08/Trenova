/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
