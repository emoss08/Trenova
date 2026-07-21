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
  { label: "Customer Payments", to: "/accounting/ar/payments", icon: HandCoinsIcon },
  { label: "AR Aging", to: "/accounting/ar/aging", icon: UsersIcon },
  { label: "Open Items", to: "/accounting/ar/open-items", icon: ClipboardListIcon },
  { label: "Customer Ledger", to: "/accounting/ar/customer-ledger", icon: BookOpenIcon },
  { label: "Manual Journals", to: "/accounting/manual-journals", icon: FileTextIcon },
  { label: "Journal Reversals", to: "/accounting/journal-reversals", icon: Undo2Icon },
  { label: "Bank Receipts", to: "/accounting/reconciliation/bank-receipts", icon: BanknoteIcon },
  { label: "Trial Balance", to: "/accounting/reports/trial-balance", icon: BarChart3Icon },
  { label: "Balance Sheet", to: "/accounting/reports/balance-sheet", icon: ScaleIcon },
] as const;

export function AccountingQuickLinks() {
  return (
    <div className="flex flex-wrap gap-1.5">
      {QUICK_LINKS.map((link) => (
        <Link
          key={link.to}
          to={link.to}
          className="inline-flex items-center gap-1.5 rounded-full border bg-card px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <link.icon className="size-3.5" />
          {link.label}
        </Link>
      ))}
    </div>
  );
}
