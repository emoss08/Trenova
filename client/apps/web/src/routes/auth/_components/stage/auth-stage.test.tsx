import { act, cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthStage, AuthStageCanvas } from "./auth-stage";
import { useAuthStage, type AuthStageControls } from "./auth-stage-context";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  reducedMotion: false,
}));

vi.mock("./glyph-field", () => ({ GlyphField: { create: mocks.create } }));

vi.mock("motion/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("motion/react")>()),
  useReducedMotion: () => mocks.reducedMotion,
}));

const TOKENS: Record<string, string> = {
  "--brand": "oklch(0.56 0.207 258)",
  "--accent-violet": "oklch(0.55 0.185 306)",
  "--accent-teal": "oklch(0.55 0.093 182)",
  "--accent-amber": "oklch(0.55 0.112 98)",
  "--accent-rose": "oklch(0.55 0.17 2)",
  "--auth-canvas": "oklch(1 0 0)",
  "--font-mono": '"Geist Mono", monospace',
};

function fakeField() {
  return {
    set: vi.fn(),
    setDone: vi.fn(),
    burst: vi.fn(),
    pause: vi.fn(),
    resume: vi.fn(),
    destroy: vi.fn(),
  };
}

let hidden = false;

function Probe({ onControls }: { onControls: (controls: AuthStageControls) => void }) {
  onControls(useAuthStage());
  return null;
}

function renderStage() {
  const captured: { controls?: AuthStageControls } = {};
  const view = render(
    <AuthStage>
      <Probe
        onControls={(controls) => {
          captured.controls = controls;
        }}
      />
      <AuthStageCanvas />
    </AuthStage>,
  );
  return { ...view, controls: () => captured.controls as AuthStageControls };
}

describe("AuthStage", () => {
  beforeEach(() => {
    mocks.create.mockReset();
    mocks.reducedMotion = false;
    hidden = false;
    Object.defineProperty(document, "hidden", { configurable: true, get: () => hidden });
    vi.spyOn(window, "getComputedStyle").mockReturnValue({
      getPropertyValue: (property: string) => TOKENS[property] ?? "",
    } as CSSStyleDeclaration);
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("creates the field from the theme's tokens and destroys it on unmount", async () => {
    const field = fakeField();
    mocks.create.mockReturnValue(field);

    const { container, unmount } = renderStage();

    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    const [canvas, options] = mocks.create.mock.calls[0];
    expect(canvas).toBe(container.querySelector("canvas"));
    expect(container.querySelector("canvas")).toHaveAttribute("aria-hidden", "true");
    expect(options).toMatchObject({
      motion: true,
      background: "#ffffff",
      fontFamily: '"Geist Mono", monospace',
    });
    expect(options.palette).toHaveLength(4);
    for (const stop of options.palette) {
      expect(stop).toMatch(/^#[0-9a-f]{6}$/);
    }

    unmount();
    expect(field.destroy).toHaveBeenCalledTimes(1);
  });

  it("pauses the loop while the tab is hidden and resumes when it returns", async () => {
    const field = fakeField();
    mocks.create.mockReturnValue(field);
    renderStage();
    await waitFor(() => expect(mocks.create).toHaveBeenCalled());

    hidden = true;
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(field.pause).toHaveBeenCalledTimes(1);

    hidden = false;
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(field.resume).toHaveBeenCalledTimes(1);
  });

  it("stops listening for visibility once unmounted", async () => {
    const field = fakeField();
    mocks.create.mockReturnValue(field);
    const { unmount } = renderStage();
    await waitFor(() => expect(mocks.create).toHaveBeenCalled());

    unmount();
    hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));

    expect(field.pause).not.toHaveBeenCalled();
  });

  it("stills the field under reduced motion", async () => {
    mocks.reducedMotion = true;
    mocks.create.mockReturnValue(fakeField());
    renderStage();

    await waitFor(() => expect(mocks.create).toHaveBeenCalled());
    expect(mocks.create.mock.calls[0][1]).toMatchObject({ motion: false });
  });

  it("forwards the controls to the field, and applies a done state set before it existed", async () => {
    const field = fakeField();
    mocks.create.mockReturnValue(field);
    const { controls } = renderStage();

    controls().setDone(true);
    await waitFor(() => expect(mocks.create).toHaveBeenCalled());
    expect(field.setDone).toHaveBeenCalledWith(true);

    controls().burst();
    expect(field.burst).toHaveBeenCalledTimes(1);
  });

  it("shows the radial wash, and keeps the controls harmless, without WebGL2", async () => {
    mocks.create.mockReturnValue(null);
    const { container, controls } = renderStage();

    await waitFor(() =>
      expect(container.querySelector(".auth-stage-fallback")).toBeInTheDocument(),
    );
    expect(() => {
      controls().burst();
      controls().setDone(true);
    }).not.toThrow();
  });
});
