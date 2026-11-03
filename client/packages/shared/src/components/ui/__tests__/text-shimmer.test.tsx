import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";

afterEach(cleanup);

function host(container: HTMLElement): HTMLElement {
  const element = container.firstElementChild;
  if (!(element instanceof HTMLElement)) throw new Error("nothing rendered");
  return element;
}

function sweep(container: HTMLElement): HTMLElement | null {
  return container.querySelector<HTMLElement>(".ui-text-shimmer-sweep");
}

describe("TextShimmer", () => {
  it("holds plain text once, with the band's copy drawn by CSS from data-text", () => {
    const { container } = render(<TextShimmer as="span">Nova is reading that…</TextShimmer>);

    expect(host(container).textContent).toBe("Nova is reading that…");
    expect(sweep(container)?.querySelector("[data-text]")?.getAttribute("data-text")).toBe(
      "Nova is reading that…",
    );
  });

  it("hides the band from assistive technology, which reads the text once", () => {
    const { container } = render(<TextShimmer as="span">Typing</TextShimmer>);

    expect(sweep(container)?.getAttribute("aria-hidden")).toBe("true");
    expect(screen.getAllByText("Typing")).toEqual([host(container)]);
  });

  it("copies rich text into the hidden band as elements", () => {
    const { container } = render(
      <TextShimmer as="span">
        Reading <b>3 invoices</b>
      </TextShimmer>,
    );

    const band = sweep(container);
    expect(band?.getAttribute("aria-hidden")).toBe("true");
    expect(band?.textContent).toBe("Reading 3 invoices");
    expect(band?.querySelector("b")?.textContent).toBe("3 invoices");
    expect(host(container).firstChild?.textContent).toBe("Reading ");
  });

  it("renders the element it is given, a paragraph by default, with the caller's attributes", () => {
    const paragraph = host(render(<TextShimmer>Loading more…</TextShimmer>).container);
    expect(paragraph.tagName).toBe("P");
    cleanup();

    const { container } = render(
      <TextShimmer as="div" className="nv-thk" aria-hidden="true">
        Typing
      </TextShimmer>,
    );
    const div = host(container);
    expect(div.tagName).toBe("DIV");
    expect(div.getAttribute("aria-hidden")).toBe("true");
    expect(div.classList).toContain("nv-thk");
    expect(div.classList).toContain("ui-text-shimmer");
  });

  it("crosses the text once in the duration given, two seconds unless told", () => {
    const quick = host(render(<TextShimmer duration={1.6}>Working</TextShimmer>).container);
    expect(quick.style.getPropertyValue("--shimmer-duration")).toBe("1.6s");
    cleanup();

    const standard = host(render(<TextShimmer>Working</TextShimmer>).container);
    expect(standard.style.getPropertyValue("--shimmer-duration")).toBe("2s");
  });

  it("renders the text plain in the same element when inactive, so a finished step stops", () => {
    const { container } = render(
      <TextShimmer as="span" className="tx" active={false}>
        Read the rate confirmation
      </TextShimmer>,
    );

    const element = host(container);
    expect(element.tagName).toBe("SPAN");
    expect(element.className).toBe("tx");
    expect(element.textContent).toBe("Read the rate confirmation");
    expect(sweep(container)).toBeNull();
    expect(element.style.getPropertyValue("--shimmer-duration")).toBe("");
  });
});
