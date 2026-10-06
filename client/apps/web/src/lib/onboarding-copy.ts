import type { OnboardingStepId } from "@/lib/onboarding-form";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type NovaSegment = { text: string; bold: boolean };

const MARK = "\u0000";

/**
 * Stand-ins handed to `t` for the values Nova says in bold. A translation may move a
 * placeholder anywhere in the sentence, so the bold runs are found by these marks in
 * the translated text rather than by position in the English.
 */
export const BOLD_MARKS = [`${MARK}0${MARK}`, `${MARK}1${MARK}`, `${MARK}2${MARK}`] as const;

/** Splits a line translated with `BOLD_MARKS` into plain and bold runs. */
export function boldSegments(translated: string, values: readonly string[]): NovaSegment[] {
  const parts = translated.split(new RegExp(`${MARK}(\\d)${MARK}`));
  const segments: NovaSegment[] = [];
  parts.forEach((part, index) => {
    if (index % 2 === 0) {
      if (part) {
        segments.push({ text: part, bold: false });
      }
      return;
    }
    const value = values[Number(part)] ?? "";
    if (value) {
      segments.push({ text: value, bold: true });
    }
  });
  return segments;
}

export function plainSegments(text: string): NovaSegment[] {
  return [{ text, bold: false }];
}

export function segmentsText(segments: readonly NovaSegment[]): string {
  return segments.map((segment) => segment.text).join("");
}

export type NovaContext = {
  firstName: string;
  company: string;
  browserZoneLabel: string;
};

/** What Nova says on each turn. Every line is fixed copy; nothing here is generated. */
export function novaLine(t: TranslateFn, id: OnboardingStepId, context: NovaContext) {
  const { firstName, company, browserZoneLabel } = context;
  switch (id) {
    case "name":
      return plainSegments(
        `${
          firstName
            ? t(
                "Hi {0}, I'm Nova. I'll help you set up your Trenova workspace. It takes about a minute, and you can change anything later.",
                firstName,
              )
            : t(
                "Hi, I'm Nova. I'll help you set up your Trenova workspace. It takes about a minute, and you can change anything later.",
              )
        }\n\n${t("To start, what's your company called?")}`,
      );
    case "timezone":
      return browserZoneLabel
        ? boldSegments(
            t(
              "Got it, {0}. Which timezone should dispatch, appointments and reports use? Your browser is set to {1}.",
              ...BOLD_MARKS,
            ),
            [company, browserZoneLabel],
          )
        : boldSegments(
            t(
              "Got it, {0}. Which timezone should dispatch, appointments and reports use?",
              ...BOLD_MARKS,
            ),
            [company],
          );
    case "address":
      return boldSegments(
        t(
          "Where is {0} headquartered? This becomes the default origin on new shipments.",
          ...BOLD_MARKS,
        ),
        [company],
      );
    case "ids":
      return plainSegments(
        t("Do you have a SCAC code or USDOT number? Both are optional — add them now or skip."),
      );
    case "operation":
      return boldSegments(
        t("How does {0} move freight? This decides which parts of Trenova lead.", ...BOLD_MARKS),
        [company],
      );
    case "sample-data":
      return plainSegments(
        t(
          "Want me to load sample data? Every screen will have something to show, and you can delete any of it at any time.",
        ),
      );
    case "review":
      return plainSegments(
        firstName
          ? t(
              "That's everything, {0}. Please check everything below before I create your workspace. If anything's wrong, click the line to fix it.",
              firstName,
            )
          : t(
              "That's everything. Please check everything below before I create your workspace. If anything's wrong, click the line to fix it.",
            ),
      );
  }
}

/** The first word of a person's name, or "" when the session does not know it. */
export function firstNameOf(fullName: string | null | undefined): string {
  return (fullName ?? "").trim().split(/\s+/)[0] ?? "";
}
