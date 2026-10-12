import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { useInView, type InViewOptions } from "../use-in-view";

function Probe(options: InViewOptions) {
  const [ref, inView] = useInView<HTMLDivElement>(options);
  return (
    <div ref={ref} data-testid="probe">
      {inView ? "in view" : "out of view"}
    </div>
  );
}

describe("useInView", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("counts everything as in view where the browser cannot tell", () => {
    vi.stubGlobal("IntersectionObserver", undefined);
    render(<Probe />);

    expect(screen.getByTestId("probe")).toHaveTextContent("in view");
  });

  describe("with an observer", () => {
    beforeEach(() => {
      installIntersectionObserver();
    });

    it("follows the element in and out of view, watching with the margin it was given", () => {
      render(<Probe rootMargin="320px 0px" />);
      const probe = screen.getByTestId("probe");

      expect(probe).toHaveTextContent("out of view");
      expect(FakeIntersectionObserver.instances.at(-1)?.options.rootMargin).toBe("320px 0px");

      act(() => FakeIntersectionObserver.show(probe));
      expect(probe).toHaveTextContent("in view");

      act(() => FakeIntersectionObserver.show(probe, false));
      expect(probe).toHaveTextContent("out of view");
    });

    it("stays in view once seen, and stops watching, when asked to look once", () => {
      render(<Probe once />);
      const probe = screen.getByTestId("probe");

      act(() => FakeIntersectionObserver.show(probe));
      expect(probe).toHaveTextContent("in view");
      expect(FakeIntersectionObserver.instances.every((observer) => observer.disconnected)).toBe(
        true,
      );

      act(() => FakeIntersectionObserver.show(probe, false));
      expect(probe).toHaveTextContent("in view");
    });
  });
});
