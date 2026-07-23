export type InspectorGridRow = [string, string];

export default function InspectorGrid({ rows }: { rows: InspectorGridRow[] }) {
  return (
    <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
      {rows.map(([label, value]) => (
        <div key={label} className="rounded-md border bg-background p-3">
          <div className="text-xs text-muted-foreground">{label}</div>
          <div className="mt-1 font-mono text-sm wrap-break-word">{value || "-"}</div>
        </div>
      ))}
    </div>
  );
}
