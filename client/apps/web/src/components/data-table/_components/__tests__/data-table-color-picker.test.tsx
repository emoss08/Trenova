import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DataTableColorPicker } from "../data-table-color-picker";

const { toastError } = vi.hoisted(() => ({ toastError: vi.fn() }));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: { error: toastError } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

async function openPicker(
  color: string | undefined,
  onChange = vi.fn().mockResolvedValue(undefined),
) {
  const user = userEvent.setup();
  render(<DataTableColorPicker color={color} label="Code" onChange={onChange} />);
  await user.click(screen.getByRole("button", { name: "Change the Code color" }));
  return { user, onChange };
}

describe("DataTableColorPicker", () => {
  it("saves the colour picked from the swatches", async () => {
    const { user, onChange } = await openPicker("#ff75c3");

    expect(screen.getByRole("button", { name: "Color #ff75c3" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await user.click(screen.getByRole("button", { name: "Color #2e8b57" }));

    expect(onChange).toHaveBeenCalledExactlyOnceWith("#2e8b57");
  });

  it("takes the colour away", async () => {
    const { user, onChange } = await openPicker("#ff75c3");

    await user.click(screen.getByRole("button", { name: "No color" }));

    expect(onChange).toHaveBeenCalledExactlyOnceWith(null);
  });

  it("saves nothing when the colour picked is the one it has", async () => {
    const { user, onChange } = await openPicker("#FF75C3");

    await user.click(screen.getByRole("button", { name: "Color #ff75c3" }));

    expect(onChange).not.toHaveBeenCalled();
  });

  it("offers a colour to a record that has none, with nothing to take away", async () => {
    const { user, onChange } = await openPicker(undefined);

    expect(screen.queryByRole("button", { name: "No color" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Color #ffd700" }));

    expect(onChange).toHaveBeenCalledExactlyOnceWith("#ffd700");
  });

  it("says when the colour could not be saved, and can be tried again", async () => {
    const onChange = vi.fn().mockRejectedValue(new Error("Color must be a valid hex color."));
    const { user } = await openPicker("#ff75c3", onChange);

    await user.click(screen.getByRole("button", { name: "Color #2e8b57" }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith("Update failed", {
        description: "Color must be a valid hex color.",
      }),
    );
    expect(screen.getByRole("button", { name: "Change the Code color" })).not.toBeDisabled();
  });
});
