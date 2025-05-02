import { createRoot } from "react-dom/client";
import { scan } from "react-scan"; // must be imported before React and React DOM
import App from "./App.tsx";
import { Providers } from "./components/providers.tsx";
import { APP_ENV } from "./constants/env.ts";
import "./styles/app.css";

createRoot(document.getElementById("root")!).render(
  // <StrictMode>
  <Providers>
    <App />
  </Providers>,
  // </StrictMode>,
);

scan({
  enabled: APP_ENV !== "production",
});
