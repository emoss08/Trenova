import { LazyMotion, domAnimation } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { Providers } from "./providers";
import { router } from "./router";
import "./styles/app.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LazyMotion features={domAnimation}>
      <Providers>
        <RouterProvider router={router} />
      </Providers>
    </LazyMotion>
  </StrictMode>,
);
