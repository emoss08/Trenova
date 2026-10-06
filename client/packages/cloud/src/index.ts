import { createPermissionLoader } from "@/lib/route-permission";
import { defineEdition } from "@trenova/edition";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { redirect, type LoaderFunction } from "react-router";
import { CloudAppBanner } from "./components/cloud-app-banner";
import { SignupPrompt } from "./components/signup-prompt";
import { usePlanRestrictions } from "./hooks/use-cloud-plan";
import { PLAN_USAGE_PATH } from "./lib/free-demo";
import { cloudPlanLimitCopy } from "./lib/plan-limit-copy";
import { loadPlanRestrictions } from "./lib/plan-restrictions";
import { signupRouteRedirect } from "./lib/signup-gate";
import { SUPPORT_ACCESS_ADMIN_PATH, SUPPORT_CONSOLE_PATH } from "./lib/support-access";

// Signup exists only on Trenova Cloud with signup switched on; everywhere else the
// address falls back to sign-in rather than showing a form the server would refuse.
const signupLoader: LoaderFunction = async () => {
  const target = await signupRouteRedirect();
  return target ? redirect(target) : null;
};

/**
 * Trenova Cloud: self-serve signup behind Turnstile, the free demo's trial banner and
 * plan page, the demo's wording for plan-limit refusals, and staff support access (the
 * support console, the session banner and the administrators' support access page). Every part is still gated
 * on the server's public config at runtime, so a self-hosted install built with this
 * overlay behaves as one built without it.
 */
export default defineEdition({
  id: "cloud",
  name: "Trenova Cloud",
  routes: {
    guest: [
      {
        path: "/signup",
        loader: signupLoader,
        async lazy() {
          const { SignupPage } = await import("./routes/signup/page");
          return { Component: SignupPage };
        },
      },
      {
        // Opened from the emailed verification link. It signs the new owner in, so
        // it is guest-only for the same reason the sign-in page is.
        path: "/signup/verify",
        loader: signupLoader,
        async lazy() {
          const { SignupVerifyPage } = await import("./routes/signup/verify-page");
          return { Component: SignupVerifyPage };
        },
      },
    ],
    protected: [
      {
        // The staff console is reachable by anyone signed in on Cloud; the server decides
        // who is platform staff, and the page says so to everyone else.
        path: SUPPORT_CONSOLE_PATH,
        async lazy() {
          const { SupportConsolePage } = await import("./routes/support-console/page");
          return { Component: SupportConsolePage };
        },
      },
    ],
    admin: [
      {
        path: "plan-usage",
        loader: createPermissionLoader(Resource.Organization, Operation.Read),
        async lazy() {
          const { PlanUsagePage } = await import("./routes/admin/plan-usage/page");
          return { Component: PlanUsagePage };
        },
      },
      {
        path: "support-access",
        loader: createPermissionLoader(Resource.Organization, Operation.Read),
        async lazy() {
          const { SupportAccessPage } = await import("./routes/admin/support-access/page");
          return { Component: SupportAccessPage };
        },
      },
    ],
  },
  adminLinks: [
    {
      href: PLAN_USAGE_PATH,
      title: "Plan & usage",
      group: "Organization",
      resource: Resource.Organization,
      requiredOperation: Operation.Read,
      platformMode: "cloud",
      after: "/admin/organization-settings",
    },
    {
      href: SUPPORT_ACCESS_ADMIN_PATH,
      title: "Support access",
      group: "Organization",
      resource: Resource.Organization,
      requiredOperation: Operation.Read,
      platformMode: "cloud",
      after: PLAN_USAGE_PATH,
    },
  ],
  slots: {
    AppBanner: CloudAppBanner,
    LoginPrompt: SignupPrompt,
  },
  plan: {
    useRestrictions: usePlanRestrictions,
    loadRestrictions: loadPlanRestrictions,
    restrictedRedirect: PLAN_USAGE_PATH,
    usagePath: PLAN_USAGE_PATH,
    limitCopy: cloudPlanLimitCopy,
  },
  messages: {
    es: async () => (await import("../i18n/messages.es.json")).default,
    "zh-TW": async () => (await import("../i18n/messages.zh-TW.json")).default,
    "zh-CN": async () => (await import("../i18n/messages.zh-CN.json")).default,
  },
});
