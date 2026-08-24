export type InspectorGridRow = [string, string];

export default function InspectorGrid({ rows }: { rows: InspectorGridRow[] }) {
  return (
    <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
      {rows.map(([label, value]) => (
        <div key={label} className="bg-background rounded-md border p-3">
          <div className="text-muted-foreground text-xs">{label}</div>
          <div className="mt-1 font-mono text-sm wrap-break-word">{value || "-"}</div>
        </div>
      ))}
    </div>
  );
}
