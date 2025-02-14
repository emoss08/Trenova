/* eslint-disable @typescript-eslint/no-non-null-assertion */
import "@fontsource-variable/inter";
import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import { Providers } from "./components/providers.tsx";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <Providers>
    <App />
  </Providers>,
);
