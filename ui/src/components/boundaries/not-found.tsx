/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { useNavigate } from "react-router";

export function NotFoundPage() {
  const navigate = useNavigate();

  return (
    <section className="bg-white font-serif min-h-screen flex items-center justify-center">
      <div className="container mx-auto mb-32">
        <div className="flex justify-center">
          <div className="w-full sm:w-10/12 md:w-8/12 text-center">
            <div
              className="bg-[url(https://cdn.dribbble.com/userupload/8726278/file/original-ab1bde6f9c74de5c8961f7fe84990cd4.gif)] h-[250px] sm:h-[350px] md:h-[600px] bg-center bg-no-repeat bg-contain"
              aria-hidden="true"
            />
            <div className="mt-[-100px]">
              <h3 className="text-2xl text-black sm:text-3xl font-bold mb-4">
                Look like you&apos;re lost
              </h3>
              <p className="mb-6 text-black sm:mb-5">
                The page you are looking for is not available!
              </p>

              <Button
                variant="default"
                onClick={() => navigate("/")}
                // className="my-5 bg-green-600 hover:bg-green-700 "
              >
                Go to Home
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
