import type {
  CardCheckState,
  CardTone,
} from "@/routes/desk/_components/artifacts/record-card/record-card";
import type { StatusState } from "../../palette-parts";

/** A Desk card's check state as the palette's dot draws it: only what needs someone is coloured. */
export const CHECK_STATE: Record<CardCheckState, StatusState> = {
  ok: "neutral",
  info: "neutral",
  warn: "warn",
  bad: "bad",
};

/** A Desk card's standing tone as the palette's pill draws it. */
export const TONE_STATE: Record<CardTone, StatusState> = {
  success: "ok",
  warn: "warn",
  danger: "bad",
  progress: "neutral",
  ink: "neutral",
  muted: "neutral",
  draft: "neutral",
};
