import { InfoIcon } from "lucide-react";
import {
  describeMatch,
  enforcementLabel,
  enforcementTone,
  rulesForField,
} from "../lib/capability";
import { cn } from "../lib/utils";
import type { ResolvedCapabilityRule, ResolvedModeProfile } from "../types/shipment";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";

type CapabilityExplainerProps = {
  profile: ResolvedModeProfile | null | undefined;
  field: string;
  className?: string;
};

export function CapabilityExplainer({ profile, field, className }: CapabilityExplainerProps) {
  const rules = rulesForField(profile, field);

  if (!profile || rules.length === 0) {
    return null;
  }

  return (
    <Popover>
      <PopoverTrigger
        type="button"
        aria-label="Why does this field behave this way?"
        className={cn(
          "inline-flex size-4 items-center justify-center rounded-full text-muted-foreground",
          "transition-colors hover:text-foreground focus-visible:outline-none",
          "focus-visible:ring-2 focus-visible:ring-ring",
          className,
        )}
      >
        <InfoIcon className="size-3.5" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-96 p-0">
        <div className="border-b border-border px-4 py-3">
          <p className="text-xs font-medium">Why this field behaves this way</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Resolved from the <span className="font-medium">{profile.profileName}</span> profile,
            matched on {describeMatch(profile.candidates?.find((c) => c.selected)?.matchedOn)}.
          </p>
        </div>
        <ul className="divide-y divide-border">
          {rules.map((rule) => (
            <RuleExplanation key={rule.key} rule={rule} />
          ))}
        </ul>
      </PopoverContent>
    </Popover>
  );
}

function RuleExplanation({ rule }: { rule: ResolvedCapabilityRule }) {
  const { provenance } = rule;

  return (
    <li className="px-4 py-3">
      <div className="flex items-baseline justify-between gap-2">
        <p className="text-xs font-medium">{rule.label}</p>
        <span className={cn("text-[11px] font-medium", enforcementTone(rule.enforcement))}>
          {enforcementLabel(rule.enforcement)}
        </span>
      </div>

      <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
        {provenance.rationale}
      </p>

      {provenance.overridden && (
        <div className="mt-2 rounded-md bg-muted px-2.5 py-2">
          <p className="text-[11px] font-medium">
            Your organization changed this from {enforcementLabel(provenance.defaultEnforcement)}
          </p>
          {provenance.overrideReason && (
            <p className="mt-0.5 text-[11px] text-muted-foreground">
              {provenance.overrideReason}
            </p>
          )}
        </div>
      )}
    </li>
  );
}
