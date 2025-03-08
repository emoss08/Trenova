import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import { Resource } from "./audit-entry";

/**
 * Route information structure for navigation
 */
export interface RouteInfo {
  key: Resource;
  label: string;
  icon?: IconDefinition;
  link?: string;
  supportsModal?: boolean;
  tree?: RouteInfo[];
  isDefault?: boolean;
}

/**
 * Command route information for command palette
 */
export interface CommandRouteInfo {
  id: string;
  link: string;
  label: string;
  icon?: IconDefinition;
}

/**
 * Command group information for command palette
 */
export interface CommandGroupInfo {
  id: string;
  label: string;
  routes: CommandRouteInfo[];
}

export interface PageInfo {
  path: string;
  supportsModal: boolean;
}

// Quick lookup for routes by resource type
export type ResourceType = string;

export interface ResourcePageInfo {
  path: string;
  supportsModal: boolean;
}
