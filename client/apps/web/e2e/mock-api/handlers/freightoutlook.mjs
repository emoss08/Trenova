// The freight outlook's operations (`/dispatch/outlook`): the outlook, one area of it, sending a
// driver toward freight and calling that off, and the page tour's record. A send is kept in the
// state, so the next read shows the driver on the way and no longer offers the move.
import {
  buildFreightArea,
  buildFreightOutlook,
  seedPositionings,
} from "../fixtures/freight-outlook.mjs";
import { TriageError } from "./triage-model.mjs";

const clock = () => Math.floor(Date.now() / 1000);

function outlookState(state) {
  state.freightOutlook ??= {
    positionings: seedPositionings(clock()),
    // The tour opens by itself only for someone who has not seen it; the fixture user has.
    tours: [
      { tour: "freight-outlook", tourVersion: 1, outcome: "Finished", seenAt: clock() - 86_400 },
    ],
  };
  return state.freightOutlook;
}

const notFound = (message) => new TriageError(message, { code: "NOT_FOUND", type: "not-found" });

export const FREIGHT_OUTLOOK_HANDLERS = {
  DispatchFreightOutlook: (state) => ({
    dispatchFreightOutlook: buildFreightOutlook(clock(), outlookState(state).positionings),
  }),

  DispatchFreightArea: (state, { cell, slotStart = null }) => {
    const area = buildFreightArea(clock(), outlookState(state).positionings, cell, slotStart);
    if (!area) throw notFound("That area is not in the outlook.");
    return { dispatchFreightArea: area };
  },

  DispatchPositionDriver: (state, { input }) => {
    const held = outlookState(state);
    const now = clock();
    const move = buildFreightOutlook(now, held.positionings).moves.find(
      (row) =>
        row.kind === "Send" && row.workerId === input.workerId && row.toCell === input.toCell,
    );
    if (!move) throw notFound("That driver is no longer offered for this area.");
    const positioning = {
      id: `dpo_outlook_${held.positionings.length + 1}`,
      workerId: move.workerId,
      workerName: move.workerName,
      fromLabel: move.fromLabel,
      toCell: move.toCell,
      toLabel: move.toLabel,
      slotStart: input.slotStart,
      freeAt: move.freeAt,
      arriveAt: move.arriveAt,
      miles: move.miles,
      valueCents: move.netCents,
      status: "Active",
      createdAt: now,
    };
    held.positionings.push(positioning);
    return { dispatchPositionDriver: positioning };
  },

  DispatchCallOffPositioning: (state, { id }) => {
    const positioning = outlookState(state).positionings.find((row) => row.id === id);
    if (!positioning) throw notFound("That send was already called off.");
    positioning.status = "Canceled";
    return { dispatchCallOffPositioning: positioning };
  },

  MyPageTourViews: (state) => ({ myPageTourViews: outlookState(state).tours }),

  RecordMyPageTour: (state, { input }) => {
    const held = outlookState(state);
    const view = {
      tour: input.tour,
      tourVersion: input.tourVersion,
      outcome: input.outcome,
      seenAt: clock(),
    };
    held.tours = [...held.tours.filter((row) => row.tour !== input.tour), view];
    return { recordMyPageTour: view };
  },
};
