import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

/** A marker class whose `map` setter throws on detach, as Google's does once the map's element is gone. */
class BrokenMarker {
  attached: unknown = null;

  get map() {
    return this.attached;
  }

  set map(value: unknown) {
    if (value === null) {
      throw new TypeError("Cannot read properties of undefined (reading 'getRootNode')");
    }
    if (value === "bad") {
      throw new Error("cannot attach");
    }
    this.attached = value;
  }
}

const library = { AdvancedMarkerElement: BrokenMarker };

const usable = { current: library as { AdvancedMarkerElement: new () => unknown } };

/** A map that tells its listeners when its tiles have loaded. */
const map = { loaded: [] as Array<() => void> };

vi.mock("@vis.gl/react-google-maps", () => ({
  AdvancedMarker: () => <i data-testid="marker" />,
  useMapsLibrary: () => usable.current,
  useMap: () => map,
}));

describe("AdvancedMarker", () => {
  beforeEach(() => {
    vi.resetModules();
    usable.current = library;
    map.loaded = [];
    vi.stubGlobal("google", {
      maps: {
        event: {
          addListenerOnce: (_map: unknown, _name: string, heard: () => void) => {
            map.loaded.push(heard);

            return { remove: () => undefined };
          },
        },
      },
    });
  });

  it("is put on a map only once that map has drawn its tiles", async () => {
    const { AdvancedMarker } = await import("../advanced-marker");
    const { queryByTestId, findByTestId } = render(
      <AdvancedMarker position={{ lat: 1, lng: 2 }} />,
    );

    expect(queryByTestId("marker")).toBeNull();
    act(() => map.loaded.forEach((heard) => heard()));

    expect(await findByTestId("marker")).toBeTruthy();
  });

  it("draws nothing, rather than throwing, while Google cannot make a marker", async () => {
    usable.current = {
      AdvancedMarkerElement: class {
        constructor() {
          throw new TypeError("Cannot read properties of undefined (reading 'keys')");
        }
      },
    };
    const { AdvancedMarker } = await import("../advanced-marker");

    const { queryByTestId } = render(<AdvancedMarker position={{ lat: 1, lng: 2 }} />);

    expect(queryByTestId("marker")).toBeNull();
  });

  it("makes taking any marker off its map safe, and leaves putting one on a map loud", async () => {
    const { AdvancedMarker } = await import("../advanced-marker");
    render(<AdvancedMarker position={{ lat: 1, lng: 2 }} />);

    const marker = new BrokenMarker();
    marker.map = "a map";
    expect(marker.map).toBe("a map");
    expect(() => {
      marker.map = null;
    }).not.toThrow();
    expect(() => {
      marker.map = "bad";
    }).toThrow("cannot attach");
  });
});
