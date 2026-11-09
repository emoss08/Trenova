import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { InfoTooltip } from "@/components/info-tooltip";

afterEach(cleanup);

describe("InfoTooltip", () => {
  it("stays reachable inside a read-only form's disabled fieldset", () => {
    render(
      <fieldset disabled>
        <InfoTooltip label="Identity">The name prints as the payee.</InfoTooltip>
      </fieldset>,
    );

    const icon = screen.getByRole("img", { name: "About Identity" });
    expect(icon.matches(":disabled")).toBe(false);
    expect(icon.tabIndex).toBe(0);
    icon.focus();
    expect(document.activeElement).toBe(icon);
  });
});
