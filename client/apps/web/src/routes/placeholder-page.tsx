import { useLocation } from "react-router";

export function PlaceholderPage() {
  const location = useLocation();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8">
      <div className="text-center">
        <h1 className="text-foreground text-2xl font-semibold">
          {formatPathToTitle(location.pathname)}
        </h1>
        <p className="text-muted-foreground mt-2">This page is under construction</p>
        <code className="bg-muted mt-4 block rounded px-3 py-1.5 text-sm">{location.pathname}</code>
      </div>
    </div>
  );
}

function formatPathToTitle(path: string): string {
  return path
    .split("/")
    .filter(Boolean)
    .map((segment) =>
      segment
        .split("-")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" "),
    )
    .join(" → ");
}
