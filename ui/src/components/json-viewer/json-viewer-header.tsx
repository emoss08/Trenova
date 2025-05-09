export function JsonCodeDiffHeader({
  title,
  lines,
}: {
  title: string;
  lines: number;
}) {
  return (
    <div className="p-2 border-b border-border bg-muted">
      <div className="flex justify-between items-center">
        <span className="text-sm font-medium text-foreground">{title}</span>
        <span className="text-xs text-muted-foreground">{lines} lines</span>
      </div>
    </div>
  );
}
