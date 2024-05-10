export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
};

export type LinkGroup = {
  key: string;
  href: string;
  title: string;
  component: React.ReactNode;
};

export type LinkGroupProps = {
  title: string;
  links: LinkGroup[];
};
