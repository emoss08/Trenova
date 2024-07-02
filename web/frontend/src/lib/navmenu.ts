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
