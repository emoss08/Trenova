import { Button } from "@trenova/shared/components/ui/button";
import { CircleDollarSign, FileCheck2, Layers, Wallet, X } from "lucide-react";
import { useState } from "react";

const STORAGE_KEY = "trenova.payroll.flow-explainer-dismissed";

const steps = [
  {
    icon: Wallet,
    title: "1. Pay accrues",
    body: "As drivers complete their moves (or the shipment reaches your pay trigger), each assigned driver earns a pay event computed from their pay profile plus any driver-specific overrides.",
  },
  {
    icon: Layers,
    title: "2. Settlements build",
    body: "At period close a batch rolls accrued events into draft settlements with deductions, advances, escrow, and guarantees — and with auto-attach on, new pay flows into open drafts as it's earned.",
  },
  {
    icon: FileCheck2,
    title: "3. Review exceptions",
    body: "Clean settlements can auto-approve; anything flagged (negative net, unusual variance, manual adjustments) waits for a reviewer.",
  },
  {
    icon: CircleDollarSign,
    title: "4. Post & pay",
    body: "Approval locks the numbers and applies side effects, posting writes the journal entry, and marking paid records how the driver was disbursed.",
  },
];

export function PayFlowExplainer() {
  const [dismissed, setDismissed] = useState(() => localStorage.getItem(STORAGE_KEY) === "true");

  if (dismissed) return null;

  return (
    <div className="relative rounded-lg border bg-muted/30 p-4">
      <Button
        size="icon"
        variant="ghost"
        className="absolute top-2 right-2 size-6"
        aria-label="Dismiss"
        onClick={() => {
          localStorage.setItem(STORAGE_KEY, "true");
          setDismissed(true);
        }}
      >
        <X className="size-3.5" />
      </Button>
      <div className="grid gap-4 sm:grid-cols-4">
        {steps.map((step) => (
          <div key={step.title} className="flex flex-col gap-1">
            <div className="flex items-center gap-1.5 text-xs font-semibold">
              <step.icon className="size-3.5 text-muted-foreground" />
              {step.title}
            </div>
            <p className="text-[11px] leading-relaxed text-muted-foreground">{step.body}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
