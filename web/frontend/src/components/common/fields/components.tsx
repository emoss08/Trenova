export function FieldDescription({ description }: { description: string }) {
  return description ? (
    <p className="text-foreground/70 text-xs">{description}</p>
  ) : null;
}
