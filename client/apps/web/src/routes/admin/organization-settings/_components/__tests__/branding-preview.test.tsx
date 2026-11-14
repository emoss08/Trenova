import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { BrandingPreview, type BrandImageSlot, type BrandMarkControl } from "../branding-preview";

function controlOf(image: "logo" | "icon", overrides: Partial<BrandMarkControl> = {}) {
  const control: BrandMarkControl = {
    uploadLabel: `Upload ${image}`,
    changeLabel: `Change ${image}`,
    onChoose: vi.fn(),
    onIntent: vi.fn(),
    onRemove: vi.fn(),
    removing: false,
    dropTarget: {
      onDragEnter: vi.fn(),
      onDragOver: vi.fn(),
      onDragLeave: vi.fn(),
      onDrop: vi.fn(),
      onPaste: vi.fn(),
    },
    dragging: false,
    ...overrides,
  };
  return control;
}

const stored =
  (src: string): BrandImageSlot["mark"] =>
  (className) => <img src={src} alt="" data-testid={src} className={className} />;

function renderPreview(logo: BrandImageSlot, icon: BrandImageSlot) {
  return render(
    <BrandingPreview
      name="Acme Logistics"
      lines={["123 Main St", "Chicago, IL 60601"]}
      logo={logo}
      icon={icon}
    />,
  );
}

describe("the sidebar's mark", () => {
  it("is the icon, cut to fill its tile, in the collapsed and the open sidebar", () => {
    renderPreview({ mark: stored("logo.webp") }, { mark: stored("icon.webp") });

    const marks = screen.getAllByTestId("icon.webp");
    expect(marks).toHaveLength(2);
    expect(marks[0]).toHaveClass("size-7", "object-cover");
    expect(screen.queryByTestId("logo.webp")).not.toBeInTheDocument();
  });

  it("is the logo, fitted whole, until there is an icon", () => {
    renderPreview({ mark: stored("logo.webp") }, {});

    expect(screen.getAllByTestId("logo.webp")[0]).toHaveClass("size-7", "object-contain");
  });

  it("is the organization's initials with neither", () => {
    renderPreview({}, {});

    expect(screen.getAllByText("AL")).toHaveLength(2);
  });

  it("goes straight to choosing an icon when pressed with none uploaded", async () => {
    const icon = controlOf("icon");
    renderPreview({ mark: stored("logo.webp") }, { control: icon });

    await userEvent.click(screen.getAllByRole("button", { name: "Upload icon" })[0]!);

    expect(icon.onChoose).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("offers to replace or remove the icon when pressed with one uploaded", async () => {
    const icon = controlOf("icon");
    renderPreview({}, { mark: stored("icon.webp"), control: icon });

    await userEvent.click(screen.getAllByRole("button", { name: "Change icon" })[0]!);
    const menu = await screen.findByRole("menu");
    await userEvent.click(within(menu).getByRole("menuitem", { name: "Remove" }));

    expect(icon.onRemove).toHaveBeenCalledTimes(1);
    expect(icon.onChoose).not.toHaveBeenCalled();
  });

  it("cannot be pressed while its picture is being removed", () => {
    renderPreview(
      {},
      { mark: stored("icon.webp"), control: controlOf("icon", { removing: true }) },
    );

    for (const mark of screen.getAllByRole("button", { name: "Change icon" })) {
      expect(mark).toBeDisabled();
    }
  });
});

describe("the invoice's logo", () => {
  async function openInvoice() {
    await userEvent.click(screen.getByRole("radio", { name: "Invoice" }));
  }

  it("is fitted whole in the masthead, beside the organization's name and address", async () => {
    renderPreview({ mark: stored("logo.webp") }, {});
    await openInvoice();

    const invoice = screen.getByRole("article", { name: "Sample invoice" });
    expect(within(invoice).getByTestId("logo.webp")).toHaveClass("max-h-11", "object-contain");
    expect(invoice).toHaveTextContent("Acme Logistics");
    expect(invoice).toHaveTextContent("123 Main St · Chicago, IL 60601");
  });

  it("leaves a place to press where the logo goes when there is none", async () => {
    const logo = controlOf("logo");
    renderPreview({ control: logo }, {});
    await openInvoice();

    await userEvent.click(screen.getByRole("button", { name: "Upload logo" }));

    expect(logo.onChoose).toHaveBeenCalledTimes(1);
  });

  it("draws no place for a logo where it cannot be changed", async () => {
    renderPreview({}, {});
    await openInvoice();

    const invoice = screen.getByRole("article", { name: "Sample invoice" });
    expect(within(invoice).queryByRole("button")).not.toBeInTheDocument();
    expect(within(invoice).queryByRole("img")).not.toBeInTheDocument();
  });
});
