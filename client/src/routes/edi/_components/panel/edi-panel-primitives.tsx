import { cn } from "@/lib/utils";

export function EDIEmptyState({ message, className }: { message: string; className?: string }) {
  return (
    <div
      className={cn(
        "rounded-md border border-dashed bg-muted/20 px-3 py-6 text-center text-sm text-muted-foreground",
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
    <section className="rounded-md border bg-muted/20 p-3">
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
      <div className="text-xs text-muted-foreground">{label}</div>
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
}: {
  label: string;
  value: React.ReactNode;
  hint?: string;
}) {
  return (
    <div className="rounded-md border bg-background p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-semibold">{value}</div>
      {hint && <div className="mt-0.5 text-[10px] text-muted-foreground">{hint}</div>}
    </div>
  );
}

export function EDIRawContent({ content }: { content: string }) {
  return (
    <pre className="max-h-72 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs whitespace-pre-wrap">
      {content}
    </pre>
  );
}
