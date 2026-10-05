import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";
import { expect, waitFor } from "storybook/test";
import { DemoBanner } from "./demo-banner";

const meta = {
  title: "Plan limit/Demo banner",
  component: DemoBanner,
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
} satisfies Meta<typeof DemoBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The digit each column of an odometer shows through its window, read from layout. */
function shownDigits(odometer: HTMLElement): string {
  const window = odometer.getBoundingClientRect();
  const middle = (window.top + window.bottom) / 2;
  return [...odometer.querySelectorAll<HTMLElement>(".tdb-digit")]
    .map((column) => {
      const face = [...column.children].find((candidate) => {
        const box = candidate.getBoundingClientRect();
        const inside = box.left >= window.left - 0.5 && box.right <= window.right + 0.5;
        return inside && Math.abs((box.top + box.bottom) / 2 - middle) < 2;
      });
      return face?.textContent ?? "?";
    })
    .join("");
}

/** A count of two digits shows both, side by side, once it has rolled to its value. */
export const TwoDigitCount: Story = {
  args: { messages: [{ kind: "shipments", value: 12 }], showPlanLink: true },
  play: async ({ canvasElement }) => {
    const odometer = await waitFor(() => {
      const node = canvasElement.querySelector<HTMLElement>(
        '.tdb-phrase[data-phase="in"] .tdb-odo',
      );
      if (!node) {
        throw new Error("the odometer has not entered yet");
      }
      return node;
    });
    await waitFor(() => expect(shownDigits(odometer)).toBe("12"), { timeout: 4000 });
  },
};

/** A single-digit count rolls to its value. */
export const OneDigitCount: Story = {
  args: { messages: [{ kind: "days", value: 7 }], showPlanLink: true },
  play: async ({ canvasElement }) => {
    const odometer = await waitFor(() => {
      const node = canvasElement.querySelector<HTMLElement>(
        '.tdb-phrase[data-phase="in"] .tdb-odo',
      );
      if (!node) {
        throw new Error("the odometer has not entered yet");
      }
      return node;
    });
    await waitFor(() => expect(shownDigits(odometer)).toBe("7"), { timeout: 4000 });
  },
};
