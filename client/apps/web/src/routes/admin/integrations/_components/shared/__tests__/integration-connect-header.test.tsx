import { cleanup, render, screen } from "@testing-library/react";
import { Dialog, DialogContent } from "@trenova/shared/components/ui/dialog";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import {
  IntegrationConnectHeader,
  type IntegrationLogo,
  ThemedLogo,
} from "../integration-connect-header";

afterEach(cleanup);

function renderHeader({
  name = "Resend",
  logo,
  description = "Create an API key and webhook in Resend.",
}: {
  name?: string;
  logo: IntegrationLogo;
  description?: ReactNode;
}) {
  return render(
    <Dialog open>
      <DialogContent>
        <IntegrationConnectHeader
          name={name}
          logo={logo}
          title="Connect with Resend"
          description={description}
        />
      </DialogContent>
    </Dialog>,
  );
}

function vendorLogos(): HTMLImageElement[] {
  // The first tile is Trenova's mark; the last is the vendor's.
  const tiles = document.querySelectorAll('[data-slot="integration-logo"]');
  const vendorTile = tiles[tiles.length - 1];
  return Array.from(vendorTile?.querySelectorAll("img") ?? []);
}

function classes(element: Element): string[] {
  return element.className.split(/\s+/);
}

describe("IntegrationConnectHeader", () => {
  it("names and describes the dialog with its title and description", () => {
    renderHeader({ logo: { light: "/logos/resend-light.svg", dark: "/logos/resend-dark.svg" } });

    const dialog = screen.getByRole("dialog");
    expect(dialog.getAttribute("aria-labelledby")).toBeTruthy();
    expect(document.getElementById(dialog.getAttribute("aria-labelledby") ?? "")?.textContent).toBe(
      "Connect with Resend",
    );
    expect(
      document.getElementById(dialog.getAttribute("aria-describedby") ?? "")?.textContent,
    ).toBe("Create an API key and webhook in Resend.");
  });

  it("keeps the logos out of the accessibility tree, since the title names the vendor", () => {
    renderHeader({ logo: { light: "/logos/resend-light.svg", dark: "/logos/resend-dark.svg" } });

    expect(screen.queryAllByRole("img")).toEqual([]);
  });

  it("shows the light-background logo outside the dark theme and the dark one inside it", () => {
    renderHeader({ logo: { light: "/logos/resend-light.svg", dark: "/logos/resend-dark.svg" } });

    const [light, dark] = vendorLogos();
    expect(vendorLogos()).toHaveLength(2);
    expect(light?.getAttribute("src")).toBe("/logos/resend-light.svg");
    expect(classes(light!)).toContain("dark:hidden");
    expect(classes(light!)).not.toContain("hidden");
    expect(dark?.getAttribute("src")).toBe("/logos/resend-dark.svg");
    expect(classes(dark!)).toContain("hidden");
    expect(classes(dark!)).toContain("dark:block");
  });

  it("draws a vendor's only logo in both themes", () => {
    renderHeader({ name: "Google Maps", logo: { light: "/logos/google-maps.svg" } });

    const logos = vendorLogos();
    expect(logos).toHaveLength(1);
    expect(logos[0]?.getAttribute("src")).toBe("/logos/google-maps.svg");
    expect(classes(logos[0]!)).not.toContain("hidden");
    expect(classes(logos[0]!)).not.toContain("dark:hidden");
  });

  it("draws a dark-only logo in both themes too", () => {
    renderHeader({ name: "WEX", logo: { dark: "/logos/wex-dark.svg" } });

    const logos = vendorLogos();
    expect(logos).toHaveLength(1);
    expect(logos[0]?.getAttribute("src")).toBe("/logos/wex-dark.svg");
    expect(classes(logos[0]!)).not.toContain("hidden");
  });

  it("draws the vendor's initials when it has no logo", () => {
    renderHeader({ name: "Comdata", logo: {} });

    const tiles = document.querySelectorAll('[data-slot="integration-logo"]');
    expect(tiles).toHaveLength(2);
    expect(tiles[1]?.textContent).toBe("CO");
    expect(vendorLogos()).toEqual([]);
  });
});

describe("ThemedLogo", () => {
  it("names both variants with the same alt text so either reads the same", () => {
    render(
      <ThemedLogo
        logo={{ light: "/logos/xero-light.svg", dark: "/logos/xero-dark.svg" }}
        alt="Xero logo"
      />,
    );

    expect(screen.getAllByAltText("Xero logo").map((image) => image.getAttribute("src"))).toEqual([
      "/logos/xero-light.svg",
      "/logos/xero-dark.svg",
    ]);
  });

  it("draws nothing when the vendor has no logo", () => {
    const { container } = render(<ThemedLogo logo={{}} alt="Comdata logo" />);

    expect(container.innerHTML).toBe("");
  });
});
