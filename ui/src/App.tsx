// @ts-expect-error fontsource-variable is not typed
import "@fontsource/inter";
// @ts-expect-error fontsource-variable is not typed
import "@fontsource/geist-mono";
import { RouterProvider } from "react-router";
import { router } from "./routing/router";

function App() {
  return <RouterProvider router={router} />;
}

export default App;
