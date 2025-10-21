export function DataTableContentInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col gap-1">{children}</div>;
}

export function DataTableContentFooter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex w-full items-center gap-2 pb-2 px-2">{children}</div>
  );
}

export function TableFilterConentInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex max-h-[300px] flex-col gap-2 overflow-y-auto px-4 py-2">
      {children}
    </div>
  );
}

export function TableFilterConentHeader({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center justify-between py-1 px-2">
      <p className="text-sm font-medium">{title}</p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}

export function TableFilterContentEmptyStat({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col p-2">
      <p className="font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

export function TableFilterContentOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col gap-0.5">{children}</div>;
}
