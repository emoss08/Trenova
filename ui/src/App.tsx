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
