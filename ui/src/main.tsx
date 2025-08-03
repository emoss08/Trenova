/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
