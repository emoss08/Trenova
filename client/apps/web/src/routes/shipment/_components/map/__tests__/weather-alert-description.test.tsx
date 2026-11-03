import { stubLayout } from "@/test/layout";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AlertDescription } from "../weather-alert-detail-panel";

const TEXT = "* WHAT...Heavy snow.\n* WHERE...Interstate 80 across the pass.";

let restoreLayout = () => {};
afterEach(() => {
  cleanup();
  restoreLayout();
});

function textRegion() {
  const toggle = screen.getByRole("button", { name: /full text|show less/i });
  const region = document.getElementById(toggle.getAttribute("aria-controls") ?? "");
  if (!region) throw new Error("the toggle names no text region");
  return { toggle, region };
}

describe("AlertDescription", () => {
  it("previews three lines until opened", () => {
    render(<AlertDescription text={TEXT} />);
    const { toggle, region } = textRegion();

    expect(toggle.textContent).toBe("Show full text");
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(region.style.height).toBe("3lh");
  });

  it("animates open to the text's own height, then settles at its natural height", () => {
    restoreLayout = stubLayout(180);
    render(<AlertDescription text={TEXT} />);
    const { toggle, region } = textRegion();

    fireEvent.click(toggle);
    expect(toggle.textContent).toBe("Show less");
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    expect(region.style.height).toBe("180px");

    fireEvent.transitionEnd(region);
    expect(region.style.height).toBe("auto");
  });

  it("closes from its open height back to three lines", async () => {
    restoreLayout = stubLayout(180);
    render(<AlertDescription text={TEXT} />);
    const { toggle, region } = textRegion();
    fireEvent.click(toggle);
    fireEvent.transitionEnd(region);

    fireEvent.click(toggle);
    expect(region.style.height).toBe("180px");
    await waitFor(() => expect(region.style.height).toBe("3lh"));
    expect(toggle.textContent).toBe("Show full text");
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
  });
});
