import { useT } from "@trenova/shared/i18n/use-t";
import {
  BanknoteIcon,
  BarChart3Icon,
  BookOpenIcon,
  ClipboardListIcon,
  FileTextIcon,
  HandCoinsIcon,
  ScaleIcon,
  Undo2Icon,
  UsersIcon,
} from "lucide-react";
import { Link } from "react-router";

const QUICK_LINKS = [
  { label: "Customer payments", to: "/accounting/ar/payments", icon: HandCoinsIcon },
  { label: "AR aging", to: "/accounting/ar/aging", icon: UsersIcon },
  { label: "Open items", to: "/accounting/ar/open-items", icon: ClipboardListIcon },
  { label: "Customer ledger", to: "/accounting/ar/customer-ledger", icon: BookOpenIcon },
  { label: "Manual journals", to: "/accounting/manual-journals", icon: FileTextIcon },
  { label: "Journal reversals", to: "/accounting/journal-reversals", icon: Undo2Icon },
  { label: "Bank receipts", to: "/accounting/reconciliation/bank-receipts", icon: BanknoteIcon },
  { label: "Trial balance", to: "/accounting/reports/trial-balance", icon: BarChart3Icon },
  { label: "Balance sheet", to: "/accounting/reports/balance-sheet", icon: ScaleIcon },
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
