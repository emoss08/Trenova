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



export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
};

type LinkGroup = {
  key: string;
  href: string;
  title: string;
  component: React.ReactNode;
};

export type LinkGroupProps = {
  title: string;
  links: LinkGroup[];
};

export type NavLinkGroup = {
  menuKey: string;
  minimizedIcon: React.ReactNode;
  label: string;
  links: LinkData[];
};

type SubLinkData = {
  key: string;
  label: string;
  link: string;
  permission?: string;
  description?: string;
  menuKey: string; // Added menuKey
};

export type LinkData = {
  key: string;
  menuKey: string; // Added menuKey
  label: string;
  link: string;
  permission?: string;
  description?: string;
  icon?: React.ReactNode;
  subLinks?: SubLinkData[];
};

export type MenuData = {
  menuKey: string;
  label: string;
  link?: string;
  content?: React.ReactNode;
  permission?: string;
  icon?: React.ReactNode;
  onClick?: () => void;
};
