import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { useForm, useWatch } from "react-hook-form";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { SettingsNested, SettingsSection } from "../settings-section";

type Values = { enabled: boolean; hours: number | null };

function Fields({ listed }: { listed: boolean }) {
  const form = useForm<Values>({ defaultValues: { enabled: true, hours: 48 } });
  const enabled = useWatch({ control: form.control, name: "enabled" });
  const fields = (
    <>
      <SwitchField
        control={form.control}
        name="enabled"
        label="Plan ahead"
        description="Plans moves before they are due."
      />
      <SettingsNested show={enabled}>
        <NumberField
          control={form.control}
          name="hours"
          label="Planning horizon"
          description="How far ahead to plan."
          sideText="hours"
        />
      </SettingsNested>
    </>
  );

  return (
    <LazyMotion features={domMax}>
      {listed ? (
        <SettingsSection title="Planning" description="How far the planner looks.">
          {fields}
        </SettingsSection>
      ) : (
        fields
      )}
    </LazyMotion>
  );
}

const row = (element: HTMLElement) => element.closest('[data-slot="setting-row"]');

// The open and close are motion; a test asserts where they end, so they end at once.
beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

describe("a settings section", () => {
  it("is a region named by its title", () => {
    render(<Fields listed />);

    const section = screen.getByRole("region", { name: "Planning" });
    expect(section).toHaveTextContent("How far the planner looks.");
  });

  // The list decides the layout: a field is a row because of where it is, so a page of
  // settings does not repeat a layout on every field.
  it("draws every field in it as a row, with its name and meaning beside its control", () => {
    render(<Fields listed />);

    const toggle = screen.getByRole("switch", { name: "Plan ahead" });
    const horizon = screen.getByRole("textbox", { name: "Planning horizon" });

    expect(row(toggle)).toHaveTextContent("Plans moves before they are due.");
    expect(row(horizon)).toHaveTextContent("How far ahead to plan.");
    expect(row(horizon)).not.toBe(row(toggle));
  });

  it("leaves a field outside a list stacked, as it is everywhere else", () => {
    render(<Fields listed={false} />);

    expect(row(screen.getByRole("switch", { name: "Plan ahead" }))).toBeNull();
    expect(row(screen.getByRole("textbox", { name: "Planning horizon" }))).toBeNull();
  });

  it("takes away the settings under a switch when it is turned off, and brings them back", async () => {
    const user = userEvent.setup();
    render(<Fields listed />);
    const toggle = screen.getByRole("switch", { name: "Plan ahead" });

    await user.click(toggle);
    await waitFor(() =>
      expect(screen.queryByRole("textbox", { name: "Planning horizon" })).toBeNull(),
    );

    await user.click(toggle);
    expect(await screen.findByRole("textbox", { name: "Planning horizon" })).toHaveValue("48");
  });
});
