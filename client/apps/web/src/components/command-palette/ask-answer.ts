import { turnCitedRecords, type TurnState } from "@/components/assistant/turn-stream";
import type { AssistantEntityRef } from "@/types/assistant";

/** The answer's words: the text the turn has streamed, without its steps. */
export function answerText(turn: TurnState): string {
  return turn.segments
    .flatMap((segment) => (segment.kind === "text" ? [segment.text] : []))
    .join("\n\n");
}

/** The words of the answer, for the clipboard, without the chips' links. */
export function plainAnswer(turn: TurnState): string {
  return answerText(turn).trim();
}

/** The records an answer was built from, as its tools returned them. */
export function answerSources(turn: TurnState | null): AssistantEntityRef[] {
  return turn ? turnCitedRecords(turn.segments) : [];
}
