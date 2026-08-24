import { cn } from "@trenova/shared/lib/utils";

export function EDIEmptyState({ message, className }: { message: string; className?: string }) {
  return (
    <div
      className={cn(
        "bg-muted/20 text-muted-foreground rounded-md border border-dashed px-3 py-6 text-center text-sm",
        className,
      )}
    >
      {message}
    </div>
  );
}

export function DetailSection({
  title,
  children,
  fullWidth,
}: {
  title: string;
  children: React.ReactNode;
  fullWidth?: boolean;
}) {
  return (
    <section className="bg-muted/20 rounded-md border p-3">
      <h3 className="mb-2 text-sm font-medium">{title}</h3>
      <div className={fullWidth ? "" : "grid grid-cols-2 gap-x-4 gap-y-2"}>{children}</div>
    </section>
  );
}

export function DetailField({
  label,
  children,
  fullWidth,
}: {
  label: string;
  children: React.ReactNode;
  fullWidth?: boolean;
}) {
  return (
    <div className={fullWidth ? "col-span-2" : ""}>
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="text-sm">{children}</div>
    </div>
  );
}

export function EDIPartnerRef({
  partner,
}: {
  partner: { code: string; name: string } | null | undefined;
}) {
  if (!partner) return <>—</>;
  return (
    <>
      {partner.code} — {partner.name}
    </>
  );
}

export function InfoTile({
  label,
  value,
  hint,
  size = "default",
  emphasizeWhenPositive = false,
}: {
  label: string;
  value: React.ReactNode;
  hint?: string;
  size?: "default" | "kpi";
  emphasizeWhenPositive?: boolean;
}) {
  const emphasized = emphasizeWhenPositive && typeof value === "number" && value > 0;
  return (
    <div className="bg-background rounded-md border p-3">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div
        className={cn(
          "mt-1 font-semibold",
          size === "kpi" ? "text-2xl leading-none tracking-tight tabular-nums" : "text-sm",
          emphasized && "text-red-600 dark:text-red-400",
        )}
      >
        {value}
      </div>
      {hint && <div className="text-muted-foreground mt-0.5 text-[10px]">{hint}</div>}
    </div>
  );
}

export function EDIRawContent({ content }: { content: string }) {
  return (
    <pre className="bg-muted/30 max-h-72 overflow-auto rounded-md border p-3 font-mono text-xs whitespace-pre-wrap">
      {content}
    </pre>
  );
}
