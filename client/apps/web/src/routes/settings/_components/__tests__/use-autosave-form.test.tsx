import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ToastOptions } from "@trenova/shared/components/ui/toast";
import type { MouseEvent } from "react";
import { useForm } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsFeedbackProvider } from "../settings-feedback";
import { useAutosaveForm } from "../use-autosave-form";

const toasts = vi.hoisted(() => ({
  success: vi.fn(),
  dismiss: vi.fn(),
  error: vi.fn(),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: toasts }));

type Values = { title: string; mode: "off" | "all" };

function Harness({ save }: { save: (next: Values, previous: Values) => Promise<void> }) {
  const form = useForm<Values>({ defaultValues: { title: "Dispatcher", mode: "off" } });
  const { commit } = useAutosaveForm({
    form,
    save,
    textFields: ["title"],
    resourceName: "Test",
  });

  return (
    <>
      <InputField control={form.control} name="title" label="Title" onBlur={commit} />
      <SegmentedField
        control={form.control}
        name="mode"
        label="Mode"
        options={[
          { value: "off", label: "Off" },
          { value: "all", label: "All" },
        ]}
      />
    </>
  );
}

function renderHarness(save: (next: Values, previous: Values) => Promise<void>) {
  return render(
    <SettingsFeedbackProvider>
      <Harness save={save} />
    </SettingsFeedbackProvider>,
  );
}

/** Presses Undo on the most recent Saved toast. */
async function pressLastUndo() {
  const call = toasts.success.mock.calls.at(-1);
  expect(call?.[0]).toBe("Saved");
  const action = (call?.[1] as ToastOptions | undefined)?.action;
  expect(action?.label).toBe("Undo");
  await act(async () => {
    action?.onClick({} as MouseEvent<HTMLButtonElement>);
  });
}

describe("useAutosaveForm", () => {
  beforeEach(() => {
    toasts.success.mockReset();
    toasts.dismiss.mockReset();
    toasts.error.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it("saves a choice the moment it is made", async () => {
    const save = vi.fn().mockResolvedValue(undefined);
    renderHarness(save);

    await userEvent.click(screen.getByRole("radio", { name: "All" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    expect(save).toHaveBeenCalledWith(
      { title: "Dispatcher", mode: "all" },
      { title: "Dispatcher", mode: "off" },
    );
    expect(toasts.success).toHaveBeenCalledTimes(1);
  });

  it("saves a text field when it loses focus, not on every keystroke", async () => {
    const save = vi.fn().mockResolvedValue(undefined);
    renderHarness(save);
    const input = screen.getByRole("textbox", { name: "Title" });

    await userEvent.clear(input);
    await userEvent.type(input, "Dispatch lead");
    expect(save).not.toHaveBeenCalled();

    await userEvent.tab();

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    expect(save.mock.calls[0][0]).toEqual({ title: "Dispatch lead", mode: "off" });
  });

  it("does not save when a field loses focus unchanged", async () => {
    const save = vi.fn().mockResolvedValue(undefined);
    renderHarness(save);

    await userEvent.click(screen.getByRole("textbox", { name: "Title" }));
    await userEvent.tab();

    expect(save).not.toHaveBeenCalled();
  });

  it("puts the previous values back and saves them when Undo is pressed", async () => {
    const save = vi.fn().mockResolvedValue(undefined);
    renderHarness(save);

    await userEvent.click(screen.getByRole("radio", { name: "All" }));
    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));

    await pressLastUndo();

    await waitFor(() => expect(save).toHaveBeenCalledTimes(2));
    expect(save.mock.calls[1]).toEqual([
      { title: "Dispatcher", mode: "off" },
      { title: "Dispatcher", mode: "all" },
    ]);
    expect(screen.getByRole("radio", { name: "Off" })).toHaveAttribute("aria-checked", "true");
  });

  it("keeps what was typed and retries the same change after a failed save", async () => {
    const save = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(undefined);
    renderHarness(save);

    await userEvent.click(screen.getByRole("radio", { name: "All" }));
    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    expect(toasts.error).toHaveBeenCalled();
    expect(screen.getByRole("radio", { name: "All" })).toHaveAttribute("aria-checked", "true");

    await userEvent.click(screen.getByRole("textbox", { name: "Title" }));
    await userEvent.tab();

    await waitFor(() => expect(save).toHaveBeenCalledTimes(2));
    expect(save.mock.calls[1][1]).toEqual({ title: "Dispatcher", mode: "off" });
  });
});
