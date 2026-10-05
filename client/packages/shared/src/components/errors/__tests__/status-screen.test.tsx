import { StatusScreen } from "@trenova/shared/components/errors/status-screen";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

afterEach(() => {
  cleanup();
});

describe("StatusScreen", () => {
  it("names no support address the build did not set", () => {
    const { container } = render(<StatusScreen>Page not found</StatusScreen>);

    expect(screen.getByText("Page not found")).toBeInTheDocument();
    expect(screen.queryByText("Need help?", { exact: false })).not.toBeInTheDocument();
    expect(container.querySelector('a[href^="mailto:"]')).toBeNull();
    expect(container.textContent).not.toMatch(/trenova\.com/);
  });

  it("offers the install's own support address when it has one", () => {
    render(<StatusScreen supportEmail="help@example.com">Page not found</StatusScreen>);

    expect(screen.getByRole("link", { name: "help@example.com" })).toHaveAttribute(
      "href",
      "mailto:help@example.com",
    );
  });

  it("keeps the reference at the foot without a support address", () => {
    render(
      <StatusScreen supportEmail="" meta="404 · Not found">
        Page not found
      </StatusScreen>,
    );

    expect(screen.getByText("404 · Not found")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /@/ })).not.toBeInTheDocument();
  });
});
