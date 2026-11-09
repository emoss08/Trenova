import { useToastSoundStore } from "@trenova/shared/components/ui/toast-sound";
import { chime } from "@yusufceng1z/ding";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@yusufceng1z/ding", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@yusufceng1z/ding")>()),
  chime: vi.fn(),
}));

afterEach(() => {
  useToastSoundStore.setState({ on: true });
  vi.mocked(chime).mockClear();
});

describe("useToastSoundStore", () => {
  it("is on until this person turns it off, and keeps the choice on the device", () => {
    expect(useToastSoundStore.getState().on).toBe(true);

    useToastSoundStore.getState().setOn(false);

    expect(useToastSoundStore.getState().on).toBe(false);
    expect(JSON.parse(window.localStorage.getItem("trenova-toast-sound") ?? "{}")).toMatchObject({
      state: { on: false },
    });
    expect(chime).not.toHaveBeenCalled();
  });

  it("plays one cue when turned back on, so the choice is heard", () => {
    useToastSoundStore.getState().setOn(false);
    useToastSoundStore.getState().setOn(true);

    expect(chime).toHaveBeenCalledExactlyOnceWith("success", 0.35);
  });
});
