import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaseAwayNotice, putAwayChoices, tomorrowMorning } from "../case-put-away";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

// A Wednesday afternoon in the test runner's own clock.
const NOW = Math.floor(new Date(2026, 10, 11, 15, 30, 0).getTime() / 1000);

afterEach(cleanup);

describe("how long a case can be put away for", () => {
  it("comes back at eight the next morning, by the reader's clock", () => {
    const back = new Date(tomorrowMorning(NOW) * 1000);

    expect([back.getDate(), back.getHours(), back.getMinutes()]).toEqual([12, 8, 0]);
  });

  it("offers the next morning, three days and a week, shortest first", () => {
    const choices = putAwayChoices(t, NOW);

    expect(choices.map((choice) => choice.label)).toEqual([
      "Until tomorrow morning",
      "For 3 days",
      "For a week",
    ]);
    expect(choices.map((choice) => choice.until - NOW)).toEqual([
      tomorrowMorning(NOW) - NOW,
      3 * 86_400,
      7 * 86_400,
    ]);
  });
});

describe("a case that is put away", () => {
  it("says so, and lets someone who may work it bring it back", async () => {
    const onBringBack = vi.fn();
    render(
      <CaseAwayNotice
        until={NOW + 86_400}
        timezone="UTC"
        canBringBack
        pending={false}
        onBringBack={onBringBack}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent(/^Put away until /);
    await userEvent.click(screen.getByRole("button", { name: "Bring it back" }));
    expect(onBringBack).toHaveBeenCalledTimes(1);
  });

  it("only says so to someone who may not work the case", () => {
    render(
      <CaseAwayNotice
        until={NOW + 86_400}
        timezone="UTC"
        canBringBack={false}
        pending={false}
        onBringBack={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "Bring it back" })).toBeNull();
  });
});
