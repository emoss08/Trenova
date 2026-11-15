import { cleanup, render, screen } from "@testing-library/react";
import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { planRestrictedResponse } from "@trenova/shared/lib/plan-limit";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import { createMemoryRouter, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

// The API's refusal of a read an add-on owns: PLAN_RESTRICTED with the feature and the
// add-on's name (pkg/errortypes NewFeatureRestrictionError).
const params = { feature: "tms.factoring", addOn: "Accounting & Cash" };

const restRefusal = new ApiRequestError(403, {
  type: "https://trenova.app/problems/plan-restricted",
  title: "Plan Restricted",
  status: 403,
  code: "PLAN_RESTRICTED",
  params,
});

const graphQLRefusal = new GraphQLRequestError({
  kind: "graphql",
  message: "This feature is part of the Accounting & Cash add-on, which is not on your plan",
  status: 200,
  graphQLErrors: [
    {
      message: "This feature is part of the Accounting & Cash add-on, which is not on your plan",
      code: "PLAN_RESTRICTED",
      type: "plan-restricted",
      params,
      extensions: { code: "PLAN_RESTRICTED", type: "plan-restricted", params },
    },
  ],
});

beforeEach(() => {
  usePermissionStore.setState({
    manifest: {
      permissions: {},
      routeAccess: {},
      unavailableFeatures: [
        {
          feature: "tms.factoring",
          featureName: "Factoring",
          addOn: "ACC",
          addOnName: "Accounting & Cash",
        },
      ],
    } as unknown as PermissionManifest,
  });
});

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("ErrorState for a read the plan refuses", () => {
  it.each([
    ["REST", restRefusal],
    ["GraphQL", graphQLRefusal],
  ])("names the add-on instead of a failure (%s)", (_transport, error) => {
    render(<ErrorState error={error} layout="section" />);

    expect(screen.getByRole("heading", { name: "Not on your plan" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "Factoring is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("You don't have access to this")).toBeNull();
    expect(screen.queryByRole("button", { name: "Try again" })).toBeNull();
    expect(document.body.textContent).not.toContain("tms.factoring");
    expect(document.body.textContent).not.toContain("PLAN_RESTRICTED");
  });

  it("says it in one row in a compact slot", () => {
    render(<ErrorState error={restRefusal} layout="compact" />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Factoring is part of the Accounting & Cash add-on, which is not on your plan.",
    );
  });
});

describe("RouteErrorBoundary for a page the plan leaves out", () => {
  it("shows the plan page, not the permission message, for the loader's refusal", async () => {
    const router = createMemoryRouter(
      [
        {
          path: "/factoring-companies",
          loader: () => {
            throw planRestrictedResponse(params);
          },
          element: <p>Factoring companies</p>,
          errorElement: <RouteErrorBoundary />,
        },
      ],
      { initialEntries: ["/factoring-companies"] },
    );
    render(<RouterProvider router={router} />);

    expect(await screen.findByRole("heading", { name: "Not on your plan" })).toBeInTheDocument();
    expect(
      screen.getByText("Ask your organization's administrator about adding it."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Go to dashboard" })).toBeInTheDocument();
    expect(screen.queryByText("You don't have access to this")).toBeNull();
    expect(screen.queryByText("403")).toBeNull();
    expect(screen.queryByText("Factoring companies")).toBeNull();
  });
});
