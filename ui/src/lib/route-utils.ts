// utils/route-utils.ts
export function generateBreadcrumbSegments(pathname: string) {
  // Remove trailing slash and split into segments
  const segments = pathname.replace(/\/$/, "").split("/").filter(Boolean);

  // Generate cumulative paths and readable labels
  return segments.map((segment, index) => {
    const path = "/" + segments.slice(0, index + 1).join("/");
    // Convert kebab-case or camelCase to Title Case
    const label = segment
      .replace(/[-_]/g, " ")
      .replace(/([A-Z])/g, " $1")
      .replace(
        /\w\S*/g,
        (txt) => txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase(),
      )
      .trim();

    return { path, label };
  });
}
