export function FieldDescription({ description }: { description: string }) {
  return description ? (
    <p className="text-xs text-foreground/70">{description}</p>
  ) : null;
}
