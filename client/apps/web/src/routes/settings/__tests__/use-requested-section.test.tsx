import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { act, cleanup, screen, waitFor } from "@testing-library/react";
import { useLocation } from "@tanstack/react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SettingsSection } from "@/lib/settings-path";
import { useRequestedSection } from "../_components/use-requested-section";

let frames: FrameRequestCallback[] = [];

beforeEach(() => {
  frames = [];
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
    frames.push(callback);
    return frames.length;
  });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => {});
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function flushFrames() {
  act(() => {
    const pending = frames;
    frames = [];
    for (const callback of pending) callback(0);
  });
}

function Probe({ jump }: { jump: (section: SettingsSection) => void }) {
  useRequestedSection(jump);
  const location = useLocation();
  return <p data-testid="where">{`${location.pathname}${location.searchStr}`}</p>;
}

async function renderAt(entry: string, jump: (section: SettingsSection) => void) {
  return await renderAtRoutes([{ path: "/settings", component: () => <Probe jump={jump} /> }], {
    initialEntries: [entry],
  });
}

describe("useRequestedSection", () => {
  it("lands on the section the address names once the page has laid out", async () => {
    const jump = vi.fn();
    await renderAt("/settings?section=notifications", jump);

    expect(jump).not.toHaveBeenCalled();
    flushFrames();

    expect(jump).toHaveBeenCalledExactlyOnceWith("notifications");
  });

  it("takes the section back out of the address and keeps every other parameter", async () => {
    await renderAt("/settings?section=security&from=email", vi.fn());

    flushFrames();

    await waitFor(() =>
      expect(screen.getByTestId("where")).toHaveTextContent("/settings?from=email"),
    );
  });

  it("drops a section it does not know without moving", async () => {
    const jump = vi.fn();
    await renderAt("/settings?section=billing", jump);

    flushFrames();

    expect(jump).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent(/^\/settings$/));
  });

  it("does nothing when the address names no section", async () => {
    const jump = vi.fn();
    await renderAt("/settings", jump);

    flushFrames();

    expect(jump).not.toHaveBeenCalled();
    expect(frames).toHaveLength(0);
  });
});
