import { useSelectOption } from "@/hooks/use-select-option";
import { endorsementTypeChoices } from "@/lib/choices";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Worker } from "@trenova/shared/types/worker";
import { useState } from "react";
import { useWatch, type Control } from "react-hook-form";

export type LicenceLineInput = {
  stateAbbreviation: string | null;
  cdlClass: string | null | undefined;
  endorsement: string | null | undefined;
  restrictions: string | null | undefined;
  licenseExpiry: number | null | undefined;
};

export type LicenceSegment = {
  key: "state" | "class" | "end" | "restr" | "exp";
  /** The card's own abbreviation before the value (CLASS, END), as printed on a CDL. */
  code: string | null;
  value: string;
  /** The endorsement in words after its letter. */
  note: string | null;
  empty: boolean;
};

const DASH = "—";

/**
 * The CDL written the way the card abbreviates it: state, class, endorsement, restrictions and
 * expiry. The abbreviations (CLASS, END, RESTR, EXP) are what the card prints in every state, so
 * they are not translated.
 */
export function licenceSegments(input: LicenceLineInput, t: TranslateFn): LicenceSegment[] {
  const endorsement = endorsementTypeChoices.find((choice) => choice.value === input.endorsement);
  const restrictions = input.restrictions?.trim().toUpperCase() ?? "";
  const expiry = input.licenseExpiry
    ? formatUnixInUserTimezone(
        input.licenseExpiry,
        { month: "2-digit", day: "2-digit", year: "numeric", timezone: "UTC" },
        DASH,
      )
    : DASH;
  return [
    {
      key: "state",
      code: null,
      value: input.stateAbbreviation || DASH,
      note: null,
      empty: !input.stateAbbreviation,
    },
    {
      key: "class",
      code: "CLASS",
      value: input.cdlClass || DASH,
      note: null,
      empty: !input.cdlClass,
    },
    {
      key: "end",
      code: "END",
      value: input.endorsement || DASH,
      note: endorsement ? t(endorsement.label).replace(/\s*\([A-Z]\)$/, "") : null,
      empty: !input.endorsement,
    },
    {
      key: "restr",
      code: "RESTR",
      value: restrictions || t("none"),
      note: null,
      empty: false,
    },
    { key: "exp", code: "EXP", value: expiry, note: null, empty: expiry === DASH },
  ];
}

/**
 * The licence line in the Licence card's header, live from the form: each segment follows its
 * field, and one that changes takes the row-changed wash.
 */
export function LicenceLine({
  control,
  storedStateId,
  storedStateAbbreviation,
}: {
  control: Control<Worker>;
  /** The issuing state saved on the record, whose abbreviation is already known. */
  storedStateId: string | null;
  storedStateAbbreviation: string | null;
}) {
  const t = useT();
  const [stateId, cdlClass, endorsement, restrictions, licenseExpiry] = useWatch({
    control,
    name: [
      "profile.licenseStateId",
      "profile.cdlClass",
      "profile.endorsement",
      "profile.cdlRestrictions",
      "profile.licenseExpiry",
    ],
  });
  const changedState = stateId && stateId !== storedStateId ? stateId : null;
  const picked = useSelectOption("US_STATE", changedState);
  const stateAbbreviation = !stateId
    ? null
    : changedState
      ? picked.option
        ? selectOptionMetaString(picked.option, "abbreviation") || null
        : null
      : storedStateAbbreviation;

  const segments = licenceSegments(
    { stateAbbreviation, cdlClass, endorsement, restrictions, licenseExpiry },
    t,
  );
  const [initial] = useState(
    () => new Map(segments.map((segment) => [segment.key, segment.value])),
  );

  return (
    <div
      aria-label={t("License as the card reads")}
      className="inline-flex h-6 max-w-full items-stretch overflow-hidden rounded-sm border bg-sunken font-mono text-xs leading-5.5 font-medium whitespace-nowrap tabular-nums"
    >
      {segments.map((segment) => {
        const changed = initial.get(segment.key) !== segment.value;
        return (
          <span
            key={changed ? `${segment.key}:${segment.value}` : segment.key}
            data-segment={segment.key}
            className={cn(
              "inline-flex items-center gap-1.5 border-l px-2 first:border-l-0",
              segment.empty && "text-foreground-subtle",
              changed && "animate-row-changed-a motion-reduce:animate-none",
            )}
          >
            {segment.code ? (
              <span className="font-normal text-foreground-subtle">{segment.code}</span>
            ) : null}
            {segment.value}
            {segment.note ? (
              <span className="ml-0.5 font-sans font-normal text-muted-foreground">
                {segment.note}
              </span>
            ) : null}
          </span>
        );
      })}
    </div>
  );
}
