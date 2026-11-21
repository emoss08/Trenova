import { cleanup, render, screen } from "@testing-library/react";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CaptureSkeleton } from "../skeleton";

const mocks = vi.hoisted(() => ({ granted: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

function grant(...resources: Resource[]) {
  for (const resource of resources) {
    mocks.granted.add(`${resource}:${Operation.Read}`);
  }
}

function outline(search = "") {
  const { container } = render(
    <TestRouter search={search}>
      <CaptureSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

describe("CaptureSkeleton", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    cleanup();
  });

  it.each([
    ["", { tabs: 3, sections: 3, rows: 5 }],
    ["?tab=profiles", { tabs: 3, sections: 1, rows: 3 }],
    ["?tab=computers", { tabs: 3, sections: 2, rows: 4 }],
  ])("draws the tab row and the tab the address opens (%s)", (search, expected) => {
    grant(Resource.DocumentControl, Resource.CaptureProfile, Resource.CaptureDevice);

    expect(outline(search)).toEqual(expected);
  });

  it("draws only the tabs this person may read, and the first of them", () => {
    grant(Resource.CaptureProfile, Resource.CaptureDevice);

    expect(outline()).toEqual({ tabs: 2, sections: 1, rows: 3 });
  });

  it("draws no tab row when only one part may be read", () => {
    grant(Resource.CaptureProfile);

    expect(outline()).toEqual({ tabs: 0, sections: 1, rows: 3 });
  });

  it("is announced once as loading", () => {
    grant(Resource.DocumentControl, Resource.CaptureProfile);
    render(
      <TestRouter>
        <CaptureSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
