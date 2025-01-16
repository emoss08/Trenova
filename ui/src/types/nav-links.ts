import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";

export type routeInfo = {
  key: string;
  label: string;
  description?: string;
  icon?: IconDefinition;
  link?: string;
  tree?: routeInfo[];
};

export type commandRouteInfo = {
  id: string;
  link: string;
  label: string;
  icon?: IconDefinition;
};

export type CommandGroupInfo = {
  id: string;
  label: string;
  routes: commandRouteInfo[];
};
