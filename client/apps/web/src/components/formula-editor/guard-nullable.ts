const REGEX_SPECIALS = /[.*+?^${}()|[\]\\]/g;

/**
 * Rewrites every bare reference to `field` in `expression` as `suggestion`
 * (typically `coalesce(field, 0)`), leaving references that already sit inside
 * a `coalesce(` call alone. Returns the expression unchanged when nothing
 * needed guarding.
 */
export function guardNullableField(expression: string, field: string, suggestion: string): string {
  const escaped = field.replace(REGEX_SPECIALS, "\\$&");
  const pattern = new RegExp(String.raw`(?<![\w.])${escaped}(?![\w.])`, "g");

  let changed = false;
  const guarded = expression.replace(pattern, (match: string, offset: number) => {
    const before = expression.slice(0, offset);
    if (/coalesce\(\s*$/.test(before)) {
      return match;
    }
    changed = true;
    return suggestion;
  });

  return changed ? guarded : expression;
}

/** Turns a server warning scope such as `breakdownDefinitions[2].expression` into a form path. */
export function scopeToFormPath(scope: string): string {
  return scope.replace(/\[(\d+)\]/g, ".$1");
}
