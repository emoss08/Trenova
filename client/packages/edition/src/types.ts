import type { IconComponent } from "@trenova/shared/components/icon-set/types";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import type { OrganizationCapabilityType } from "@trenova/shared/types/organization-capability";
import type { OperationType } from "@trenova/shared/types/permission";
import type { PlatformMode } from "@trenova/shared/types/platform";
import type { ComponentType, ReactNode } from "react";
import type { LoaderFunction, RouteObject } from "react-router";

/**
 * Where an edition route is mounted in the web app's router.
 *
 * - `public`: outside the app shell and every session gate, like the tender offer pages.
 * - `guest`: under the guest loader beside `/login`; a signed-in visitor is sent home.
 * - `protected`: inside the app shell, behind the protected loader.
 * - `admin`: inside the admin layout, as a child of `/admin` (paths are relative).
 */
export type EditionRouteMount = "public" | "guest" | "protected" | "admin";

export const EDITION_ROUTE_MOUNTS: readonly EditionRouteMount[] = [
  "public",
  "guest",
  "protected",
  "admin",
];

export type EditionRoutes = Partial<Record<EditionRouteMount, readonly RouteObject[]>>;

/** Fields an edition entry and the host's own entries share for visibility. */
type EditionVisibility = {
  /** The permission resource the entry needs read access to. */
  resource?: string;
  /** Hidden when the organization has the capability turned off. */
  capability?: OrganizationCapabilityType;
  /** Shown only on an install running in this platform mode. */
  platformMode?: PlatformMode;
};

/** An entry in the admin area's link list (`/admin/...`). */
export type EditionAdminLink = EditionVisibility & {
  /** The operation `resource` needs; read when absent. */
  requiredOperation?: OperationType;
  href: string;
  title: string;
  group?: string;
  /** The host link this one follows. Appended to the list when absent or unknown. */
  after?: string;
};

/** An entry added to one of the host's navigation modules. */
export type EditionNavItem = {
  moduleId: string;
  /** The module's group to place the item in. The module's top level when absent. */
  groupId?: string;
  /** The id of the sibling item this one follows. Appended when absent or unknown. */
  after?: string;
  item: EditionVisibility & {
    id: string;
    label: string;
    path: string;
    icon?: IconComponent;
  };
};

export type LoginPromptProps = {
  /** What the host shows under the sign-in heading when the edition has nothing to add. */
  fallback: ReactNode;
};

/**
 * Places in the host's layout an edition may render into. Every slot renders nothing
 * (or, for the login prompt, the host's fallback) when the edition leaves it out.
 */
export type EditionSlots = {
  /** Above the app shell's header, on every signed-in page. */
  AppBanner?: ComponentType;
  /** Mounted once at the router root, for dialogs and listeners that outlive a page. */
  RootHost?: ComponentType;
  /** The line under the sign-in heading, where a sign-up link belongs. */
  LoginPrompt?: ComponentType<LoginPromptProps>;
  /** The sign-in panel between the headline and the credential receipt. */
  AuthAmbient?: ComponentType;
};

/** The words the plan-limit dialog shows for one refusal. */
export type PlanLimitCopy = {
  title: string;
  description: string;
  guidance: string;
  usage: { used: number; limit: number; usedLabel: string; limitLabel: string } | null;
};

/**
 * Where the organization's plan comes from. The host owns the capability names
 * (`PlanCapability`), the nav and route plumbing that hides what a plan withholds, and
 * the dialog that explains a QUOTA_EXCEEDED / PLAN_RESTRICTED refusal; the edition
 * supplies the plan itself.
 */
export type EditionPlan = {
  /** The capabilities the plan withholds, for rendering. Must be a stable React hook. */
  useRestrictions: () => readonly string[];
  /** The same list for a route loader. Rejecting keeps the route reachable. */
  loadRestrictions: () => Promise<readonly string[]>;
  /** Where a route the plan withholds sends the visitor instead. */
  restrictedRedirect: string;
  /** The page that shows the plan and its usage, linked from the plan-limit dialog. */
  usagePath?: string;
  /** Edition copy for a refusal; null falls back to the host's generic copy. */
  limitCopy?: (notice: PlanLimitNotice, t: TranslateFn) => PlanLimitCopy | null;
};

export type EditionMessageLoader = () => Promise<Record<string, string>>;

/** Translations for the edition's own strings, keyed by English source text. */
export type EditionMessages = Partial<Record<Locale, EditionMessageLoader>>;

export type EditionDefinition = {
  /** A stable identifier, e.g. "cloud". */
  id: string;
  /** A human name for logs and diagnostics. */
  name: string;
  routes?: EditionRoutes;
  adminLinks?: readonly EditionAdminLink[];
  navItems?: readonly EditionNavItem[];
  slots?: EditionSlots;
  /**
   * Run after the session check on every protected page request, in order; the first
   * one that returns a value (a redirect) wins. They must not throw for an outage.
   */
  protectedLoaders?: readonly LoaderFunction[];
  plan?: EditionPlan;
  messages?: EditionMessages;
};

export type ResolvedEdition = {
  id: string;
  name: string;
  routes: Readonly<Record<EditionRouteMount, readonly RouteObject[]>>;
  adminLinks: readonly EditionAdminLink[];
  navItems: readonly EditionNavItem[];
  slots: Required<EditionSlots>;
  protectedLoaders: readonly LoaderFunction[];
  plan: EditionPlan;
  messages: EditionMessages;
};
