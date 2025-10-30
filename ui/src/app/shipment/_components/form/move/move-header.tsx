export function MoveInformationHeader({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex justify-between items-center p-3 border-b border-sidebar-border">
      {children}
    </div>
  );
}
