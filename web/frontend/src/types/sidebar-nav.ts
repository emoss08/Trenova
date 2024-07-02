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
