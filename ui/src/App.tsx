import { RouterProvider } from "react-router";
import PWABadge from "./PWABadge.tsx";
import { router } from "./routing/router";

function App() {
  return (
    <>
      <RouterProvider router={router} />
      <PWABadge />
    </>
  );
}

export default App;
