import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import { Providers } from "./components/providers.tsx";
import "./styles/app.css";

createRoot(document.getElementById("root")!).render(
  // <StrictMode>
  <Providers>
    <App />
  </Providers>,
  // </StrictMode>,
);
