import { InputField } from "@/components/fields/input-field";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { SettingsTabs, type SettingsTab } from "../settings-tabs";

type Values = { name: string; code: string };

function Harness({
  tabs,
  onForm,
}: {
  tabs: number;
  onForm?: (form: UseFormReturn<Values>) => void;
}) {
  const form = useForm<Values>({ defaultValues: { name: "Acme", code: "AC" } });
  const [open, setOpen] = useState("general");
  onForm?.(form);

  const all: SettingsTab<keyof Values>[] = [
    {
      value: "general",
      label: "General",
      fields: ["name"],
      content: <InputField control={form.control} name="name" label="Name" />,
    },
    {
      value: "codes",
      label: "Codes",
      fields: ["code"],
      content: <InputField control={form.control} name="code" label="Code" />,
    },
  ];

  return (
    <FormProvider {...form}>
      <SettingsTabs tabs={all.slice(0, tabs)} value={open} onValueChange={setOpen} />
    </FormProvider>
  );
}

describe("a settings form split into tabs", () => {
  it("shows one tab's settings at a time", async () => {
    const user = userEvent.setup();
    render(<Harness tabs={2} />);

    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("textbox", { name: "Name" })).toBeVisible();
    expect(screen.queryByRole("textbox", { name: "Code" })).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Codes" }));

    expect(screen.getByRole("textbox", { name: "Code" })).toBeVisible();
    expect(screen.queryByRole("textbox", { name: "Name" })).toBeNull();
  });

  // A field that is not mounted is not registered, and a server error on it would be sent
  // to the form's root instead of waiting on the tab that owns it.
  it("keeps the fields of a tab that is not open mounted", () => {
    render(<Harness tabs={2} />);

    const field = screen.getByLabelText("Code");
    expect(field).toHaveValue("AC");
    expect(field.closest('[data-slot="tabs-content"]')).toHaveAttribute("hidden");
  });

  // The tab library keeps a tab that was just left on screen until its animations end, and
  // the arriving tab's slide replayed on it: the old tab's settings showed under the new
  // one's, and stacked up when tabs were switched quickly. Whether an element is on screen
  // is the browser's layout, which this environment does not compute, so this holds the
  // rule that takes a left tab away rather than the pixels; the frame-by-frame check is in
  // the browser.
  it("takes a tab off screen the moment it is left, whichever tab that is", () => {
    render(<Harness tabs={2} />);

    for (const label of ["Name", "Code"]) {
      const panel = screen.getByLabelText(label).closest('[data-slot="tabs-content"]');
      expect(panel?.className.split(" ")).toContain("data-ending-style:hidden");
    }
  });

  it("marks the tab that holds a field to fix, and only that tab", () => {
    const held: { form?: UseFormReturn<Values> } = {};
    render(<Harness tabs={2} onForm={(form) => (held.form = form)} />);

    expect(screen.queryByText("Has fields to fix")).toBeNull();

    act(() => held.form!.setError("code", { type: "validation", message: "Code is taken" }));

    expect(
      within(screen.getByRole("tab", { name: /Codes/ })).getByText("Has fields to fix"),
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole("tab", { name: /General/ })).queryByText("Has fields to fix"),
    ).toBeNull();
  });

  // The tabs are how a person moves about the page, so they hold their place: only what
  // is under them is inside the region that scrolls.
  it("keeps the tab row out of the region that scrolls, and every tab's settings in it", () => {
    render(<Harness tabs={2} />);
    const scrolls = (element: Element) => element.closest('[data-slot="scroll-area-viewport"]');

    expect(scrolls(screen.getByRole("tablist"))).toBeNull();
    expect(scrolls(screen.getByLabelText("Name"))).not.toBeNull();
    expect(scrolls(screen.getByLabelText("Code"))).toBe(scrolls(screen.getByLabelText("Name")));
  });

  it("opens a tab at its top, not as far down as the last one was read", async () => {
    const user = userEvent.setup();
    render(<Harness tabs={2} />);
    const viewport = screen
      .getByLabelText("Name")
      .closest<HTMLElement>('[data-slot="scroll-area-viewport"]')!;
    viewport.scrollTop = 480;
    expect(viewport.scrollTop).toBe(480);

    await user.click(screen.getByRole("tab", { name: "Codes" }));

    expect(viewport.scrollTop).toBe(0);
  });

  it("draws no tab row for a page with one tab", () => {
    render(<Harness tabs={1} />);

    expect(screen.queryByRole("tablist")).toBeNull();
    expect(screen.getByRole("textbox", { name: "Name" })).toBeVisible();
  });
});
