import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
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
  return <p data-testid="where">{`${location.pathname}${location.search}`}</p>;
}

function renderAt(entry: string, jump: (section: SettingsSection) => void) {
  return render(
    <MemoryRouter initialEntries={[entry]}>
      <Routes>
        <Route path="/settings" element={<Probe jump={jump} />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("useRequestedSection", () => {
  it("lands on the section the address names once the page has laid out", () => {
    const jump = vi.fn();
    renderAt("/settings?section=notifications", jump);

    expect(jump).not.toHaveBeenCalled();
    flushFrames();

    expect(jump).toHaveBeenCalledExactlyOnceWith("notifications");
  });

  it("takes the section back out of the address and keeps every other parameter", () => {
    renderAt("/settings?section=security&from=email", vi.fn());

    flushFrames();

    expect(screen.getByTestId("where")).toHaveTextContent("/settings?from=email");
  });

  it("drops a section it does not know without moving", () => {
    const jump = vi.fn();
    renderAt("/settings?section=billing", jump);

    flushFrames();

    expect(jump).not.toHaveBeenCalled();
    expect(screen.getByTestId("where")).toHaveTextContent(/^\/settings$/);
  });

  it("does nothing when the address names no section", () => {
    const jump = vi.fn();
    renderAt("/settings", jump);

    flushFrames();

    expect(jump).not.toHaveBeenCalled();
    expect(frames).toHaveLength(0);
  });
});
