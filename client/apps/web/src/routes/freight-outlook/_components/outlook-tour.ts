import type { TourDefinition } from "@/components/tour/tour-model";

/** The parts of the page the tour points at. The brief's sections are named by their ids. */
export const OUTLOOK_TOUR_TARGET = {
  figures: "outlook-figures",
  horizon: "outlook-horizon",
  summary: "outlook-summary",
  network: "outlook-network",
  windows: "outlook-windows",
  areas: "outlook-areas",
  record: "outlook-record",
  moves: "outlook-moves",
  pace: "outlook-pace",
} as const;

/**
 * The freight outlook's tour: what the page is for, then each part in the order a dispatcher
 * reads it, top to bottom and then the brief. A part that is not drawn (no moves today, the
 * brief folded away) is passed over.
 */
export const OUTLOOK_TOUR: TourDefinition = {
  key: "freight-outlook",
  version: 1,
  steps: (t) => [
    {
      title: t("Welcome to the freight outlook"),
      body: t(
        "This page looks two days ahead and shows where trucks will be short and where drivers will sit idle, before the freight is booked. It learns from your own shipment history. The tour takes about a minute.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.figures,
      title: t("Start with the five figures"),
      body: t(
        "Loads with no truck nearby and drivers with nothing nearby are the two problems. Across the network says whether you have enough drivers overall. Worth acting on is what the suggested moves would earn. Range held is how often the forecast has been right lately.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.horizon,
      title: t("Choose how far ahead"),
      body: t("Read the next 12, 24 or 48 hours. Everything on the page follows this choice."),
    },
    {
      target: OUTLOOK_TOUR_TARGET.summary,
      title: t("The outlook in a sentence"),
      body: t(
        "The same story in words: which areas run short, how many drivers will wait, and what fixing it is worth. Select an area's name to open it.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.network,
      title: t("Loads against drivers"),
      body: t(
        "One line is the freight that still needs a driver and the other is the drivers free to take it, across your whole network. The bars underneath are the loads with no truck nearby and the drivers idle in each four-hour window.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.windows,
      title: t("Pin a window"),
      body: t(
        "Each column is a four-hour window. Select one to read the whole page at that hour, and select Show all to go back.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.areas,
      side: "top",
      title: t("One row for each area"),
      body: t(
        "A minus number is trucks short in that window and a plus number is drivers to spare. The last column is what one more driver ending there is worth. Select a row to see who books there and who can reach it in time.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.record,
      side: "left",
      title: t("How far to trust it"),
      body: t(
        "Each of the last 14 days: the range the forecast gave and the loads that came. Days outside the range are marked, so you can judge it for yourself.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.moves,
      side: "left",
      title: t("The moves that fix it"),
      body: t(
        "Ranked by what each is worth: send an empty driver, book a load that ends where trucks are short, keep a driver in an area about to get busy, or call a customer who is behind.",
      ),
    },
    {
      target: OUTLOOK_TOUR_TARGET.pace,
      side: "left",
      title: t("Customers against their usual pace"),
      body: t(
        "Who has tendered less than they usually have by now. Behind means freight you normally get has not been booked yet. You can take this tour again from the information button beside the page title.",
      ),
    },
  ],
};
