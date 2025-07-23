/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

// @ts-expect-error fontsource-variable is not typed
import "@fontsource-variable/inter"; // Defaults to wght axis
// @ts-expect-error fontsource-variable is not typed
import "@fontsource/geist-mono";
import { RouterProvider } from "react-router";
import { router } from "./routing/router";

function App() {
  return <RouterProvider router={router} />;
}

export default App;
