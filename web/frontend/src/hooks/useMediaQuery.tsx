import * as React from "react";

export function useMediaQuery(query: string) {
  const [value, setValue] = React.useState(false);

  React.useEffect(() => {
    function onChange(event: MediaQueryListEvent) {
      setValue(event.matches);
    }

    const result = matchMedia(query);
    result.addEventListener("change", onChange);
    setValue(result.matches);

    return () => result.removeEventListener("change", onChange);
  }, [query]);

  return value;
}

// Example usage:
//
// ```tsx
// import { useMediaQuery } from "./useMediaQuery";
//
// function MyComponent() {
//   const isMobile = useMediaQuery("(max-width: 768px)");
//
//   return <div>{isMobile ? "Mobile" : "Desktop"}</div>;
// }
