import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { FormSection } from "@trenova/shared/components/ui/form";

afterEach(cleanup);

describe("FormSection", () => {
  it("is a group named by its title, whatever the title holds", () => {
    render(<FormSection title="Billing & rating">fields</FormSection>);

    expect(screen.getByRole("group", { name: "Billing & rating" }).textContent).toContain("fields");
  });

  it("names each of two sections that share a title by its own heading", () => {
    render(
      <>
        <FormSection title="Address">first</FormSection>
        <FormSection title="Address">second</FormSection>
      </>,
    );

    const groups = screen.getAllByRole("group", { name: "Address" });
    expect(groups).toHaveLength(2);
    expect(groups[0].getAttribute("aria-labelledby")).not.toBe(
      groups[1].getAttribute("aria-labelledby"),
    );
  });

  it("is an unnamed group when it has no title", () => {
    render(<FormSection description="Optional details">fields</FormSection>);

    expect(screen.getByRole("group").getAttribute("aria-labelledby")).toBeNull();
  });
});
