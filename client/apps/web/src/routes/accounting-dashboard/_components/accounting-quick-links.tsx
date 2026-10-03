import { useT } from "@trenova/shared/i18n/use-t";
import {
  BankNote01Icon,
  BarChart07Icon,
  BookOpen01Icon,
  ClipboardListIcon,
  CoinsHandIcon,
  File06Icon,
  FlipBackwardIcon,
  Scales01Icon,
  Users01Icon,
} from "@trenova/shared/components/icons";
import { Link } from "react-router";

const QUICK_LINKS = [
  { label: "Customer payments", to: "/accounting/ar/payments", icon: CoinsHandIcon },
  { label: "AR aging", to: "/accounting/ar/aging", icon: Users01Icon },
  { label: "Open items", to: "/accounting/ar/open-items", icon: ClipboardListIcon },
  { label: "Customer ledger", to: "/accounting/ar/customer-ledger", icon: BookOpen01Icon },
  { label: "Manual journals", to: "/accounting/manual-journals", icon: File06Icon },
  { label: "Journal reversals", to: "/accounting/journal-reversals", icon: FlipBackwardIcon },
  { label: "Bank receipts", to: "/accounting/reconciliation/bank-receipts", icon: BankNote01Icon },
  { label: "Trial balance", to: "/accounting/reports/trial-balance", icon: BarChart07Icon },
  { label: "Balance sheet", to: "/accounting/reports/balance-sheet", icon: Scales01Icon },
] as const;

export function AccountingQuickLinks() {
  const t = useT();

  return (
    <div className="flex flex-wrap gap-1.5">
      {QUICK_LINKS.map((link) => (
        <Link
          key={link.to}
          to={link.to}
          className="bg-card text-muted-foreground hover:bg-muted hover:text-foreground inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs transition-colors"
        >
          <link.icon className="size-3.5" />
          {t(link.label)}
        </Link>
      ))}
    </div>
  );
}
