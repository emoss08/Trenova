import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { vi } from "vitest";
import { CardWhere, mapPoint } from "../card-where";

beforeEach(() => {
  installIntersectionObserver();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function draw(props: Partial<Parameters<typeof CardWhere>[0]> = {}) {
  return render(
    <CardWhere
      point={{ latitude: 32.7767, longitude: -96.797 }}
      place="Dallas yard"
      placeMissing="No known location yet"
      line="Where its last move left it"
      mapLabel="Map of Dallas yard"
      noMapLabel="No known location"
      {...props}
    />,
  );
}

describe("mapPoint", () => {
  it("is a point only when both coordinates are numbers", () => {
    expect(mapPoint(1, 2)).toEqual({ latitude: 1, longitude: 2 });
    expect(mapPoint(0, 0)).toEqual({ latitude: 0, longitude: 0 });
    expect(mapPoint(null, 2)).toBeNull();
    expect(mapPoint(1, undefined)).toBeNull();
    expect(mapPoint(Number.NaN, 2)).toBeNull();
  });
});

describe("CardWhere", () => {
  it("names the place and holds the map's square until it comes into view", async () => {
    const { container } = draw();

    expect(screen.getByText("Dallas yard")).toBeInTheDocument();
    expect(screen.getByText("Where its last move left it")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-map")).toBeNull();

    const square = container.querySelector(".dk-rec-scope");
    if (!square) throw new Error("No map square");
    act(() => FakeIntersectionObserver.show(square));

    const map = await screen.findByRole("img", { name: "Map of Dallas yard" }, { timeout: 3000 });
    expect(map.querySelectorAll("img").length).toBeGreaterThan(0);
  });

  it("shows a bare pin in the same square when there is nowhere to put the dot", () => {
    const { container } = draw({ point: null, place: null, lineOff: true });

    expect(screen.getByRole("img", { name: "No known location" })).toBeInTheDocument();
    expect(screen.getByText("No known location yet")).toHaveClass("dk-rec-unknown");
    expect(container.querySelector(".dk-rec-motion")).toHaveClass("dk-off");
  });

  it("holds the lines and leaves the square empty while the place is read", () => {
    const { container } = draw({ point: null, pending: true });

    expect(container.querySelector(".dk-rec-wait")).not.toBeNull();
    expect(screen.queryByRole("img")).toBeNull();
    expect(screen.queryByText("Dallas yard")).toBeNull();
  });

  it("marks a place that is where the record is now", () => {
    const { container } = draw({ live: true });
    expect(container.querySelector(".dk-rec-where")).toHaveClass("dk-live");
  });

  it("says a place that could not be read in its own place, with what it lists under it", () => {
    draw({ error: new Error("Telematics is down"), children: <p>Under the map</p> });

    expect(screen.getByText("Telematics is down")).toBeInTheDocument();
    expect(screen.getByText("Under the map")).toBeInTheDocument();
  });

  it("drops empty facts", () => {
    const { container } = draw({ facts: ["", "Fuel 64%"] });
    expect(container.querySelector(".dk-rec-geo")).toHaveTextContent("Fuel 64%");
    expect(container.querySelectorAll(".dk-rec-geo .dk-rec-q > span")).toHaveLength(1);
  });
});
