export function EDIEmptyState({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-3 py-6 text-center text-sm text-muted-foreground">
      {message}
    </div>
  );
}
