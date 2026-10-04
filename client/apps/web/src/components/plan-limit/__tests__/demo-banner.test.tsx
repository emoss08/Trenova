import type { CloudPlanState } from "@/hooks/use-cloud-plan";
import type { DemoBannerMessage } from "@/lib/cloud-trial";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CloudTrialBanner } from "../cloud-trial-banner";
import { DEMO_BANNER_HOLD_MS, DemoBanner } from "../demo-banner";

const cloudPlan = vi.hoisted(() => ({ current: null as unknown }));

vi.mock("@/hooks/use-cloud-plan", () => ({
  useCloudPlan: () => cloudPlan.current,
}));

const DAYS: DemoBannerMessage = { kind: "days", value: 6 };
const SHIPMENTS: DemoBannerMessage = { kind: "shipments", value: 10 };

function phrase(kind: DemoBannerMessage["kind"]) {
  const node = screen
    .getByTestId("demo-banner")
    .querySelector<HTMLElement>(`.tdb-phrase[data-kind="${kind}"]`);
  if (!node) {
    throw new Error(`no ${kind} phrase`);
  }
  return node;
}

function spoken(kind: DemoBannerMessage["kind"]) {
  const node = phrase(kind);
  const count = node.querySelector(".tdb-odo .sr-only")?.textContent ?? "";
  const words = [...node.children]
    .filter((child) => !child.classList.contains("tdb-odo"))
    .map((child) => child.textContent)
    .join(" ");
  return `${count} ${words}`;
}

function renderBanner(messages: DemoBannerMessage[], showPlanLink = true) {
  return render(
    <MemoryRouter>
      <DemoBanner messages={messages} showPlanLink={showPlanLink} />
    </MemoryRouter>,
  );
}

async function settle(ms = 0) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

describe("DemoBanner", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("opens on days left and turns to shipments left after the hold", async () => {
    renderBanner([DAYS, SHIPMENTS]);
    await settle(50);

    expect(screen.getByRole("status")).toHaveTextContent("Free demo");
    expect(phrase("days").dataset.phase).toBe("in");
    expect(spoken("days")).toBe("6 days left.");
    expect(phrase("shipments").dataset.phase).toBe("gone");
    expect(screen.getByText("After that the workspace becomes read-only.")).toBeInTheDocument();

    await settle(DEMO_BANNER_HOLD_MS);
    expect(phrase("days").dataset.phase).toBe("out");
    await settle(600);
    expect(phrase("shipments").dataset.phase).toBe("in");
    expect(spoken("shipments")).toBe("10 shipments left.");
    expect(phrase("days").dataset.phase).toBe("gone");

    await settle(DEMO_BANNER_HOLD_MS + 400);
    expect(phrase("days").dataset.phase).toBe("in");
  });

  it("rolls each digit of the count to its value", async () => {
    renderBanner([{ kind: "shipments", value: 250 }]);
    await settle(50);

    const digits = [...phrase("shipments").querySelectorAll<HTMLElement>(".tdb-digit")];
    expect(digits.map((digit) => digit.dataset.digit)).toEqual(["2", "5", "0"]);
    expect(digits.map((digit) => digit.style.transform)).toEqual([
      "translateY(-3em)",
      "translateY(-7.5em)",
      "translateY(-0em)",
    ]);
  });

  it("uses the singular for one day and one shipment", async () => {
    renderBanner([
      { kind: "days", value: 1 },
      { kind: "shipments", value: 1 },
    ]);
    await settle(50);

    expect(spoken("days")).toBe("1 day left.");
    expect(spoken("shipments")).toBe("1 shipment left.");
  });

  it("holds the current message while pointed at, and resumes after", async () => {
    renderBanner([DAYS, SHIPMENTS]);
    await settle(50);

    fireEvent.mouseEnter(screen.getByTestId("demo-banner"));
    await settle(DEMO_BANNER_HOLD_MS * 3);
    expect(phrase("days").dataset.phase).toBe("in");

    fireEvent.mouseLeave(screen.getByTestId("demo-banner"));
    await settle(DEMO_BANNER_HOLD_MS + 400);
    expect(phrase("shipments").dataset.phase).toBe("in");
  });

  it("does not turn when only one count applies", async () => {
    renderBanner([SHIPMENTS]);
    await settle(DEMO_BANNER_HOLD_MS * 3);

    expect(phrase("shipments").dataset.phase).toBe("in");
    expect(screen.getByTestId("demo-banner").querySelectorAll(".tdb-phrase")).toHaveLength(1);
  });

  it("links to Plan & usage only for someone who may open it", async () => {
    const { unmount } = renderBanner([DAYS], true);
    expect(screen.getByRole("link", { name: "Plan & usage" })).toHaveAttribute(
      "href",
      "/admin/plan-usage",
    );
    unmount();

    renderBanner([DAYS], false);
    expect(screen.queryByRole("link", { name: "Plan & usage" })).not.toBeInTheDocument();
  });
});

function planState(overrides: Partial<CloudPlanState>): CloudPlanState {
  return {
    isCloud: true,
    summary: undefined,
    trial: null,
    isLoading: false,
    isError: false,
    refetch: () => undefined,
    isFetching: false,
    ...overrides,
  };
}

describe("CloudTrialBanner", () => {
  beforeEach(() => {
    usePermissionStore.setState({ hasPermission: () => true } as never);
  });

  it("shows the countdown banner while the free demo runs", () => {
    cloudPlan.current = planState({
      trial: { kind: "trialing", daysRemaining: 4, trialEndsAt: 1 },
      summary: {
        usage: [
          {
            meterKey: "shipments.total",
            unit: "",
            limit: 12,
            used: 3,
            remaining: 9,
            windowStart: 0,
            windowEnd: 0,
          },
        ],
      } as never,
    });
    render(
      <MemoryRouter>
        <CloudTrialBanner />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("demo-banner")).toBeInTheDocument();
    expect(spoken("shipments")).toBe("9 shipments left.");
  });

  it("shows the read-only notice once the trial has ended", () => {
    cloudPlan.current = planState({
      trial: { kind: "read_only", readOnlyUntil: null, daysUntilPurge: null },
    });
    render(
      <MemoryRouter>
        <CloudTrialBanner />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("demo-banner")).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Your free demo has ended.");
  });

  it("renders nothing off the free demo", () => {
    cloudPlan.current = planState({ trial: null });
    const { container } = render(
      <MemoryRouter>
        <CloudTrialBanner />
      </MemoryRouter>,
    );

    expect(container).toBeEmptyDOMElement();
  });
});
