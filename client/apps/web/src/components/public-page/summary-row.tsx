export function SummaryRow({ label, value }: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="flex items-start justify-between gap-4 py-1.5">
      <span className="text-muted-foreground shrink-0 text-xs">{label}</span>
      <span className="text-right text-xs font-medium">{value}</span>
    </div>
  );
}
