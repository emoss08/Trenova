/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import {
  faArrowLeft,
  faArrowRight,
  faXmark,
} from "@fortawesome/pro-regular-svg-icons";
import { Portal } from "@radix-ui/react-portal";
import { useEffect, useState } from "react";
import { useTour } from "./tour-provider";

interface SpotlightStyles {
  left: number;
  top: number;
  width: number;
  height: number;
}

export function Tour() {
  const { isOpen, currentStep, steps, closeTour, nextStep, prevStep } =
    useTour();

  const [spotlight, setSpotlight] = useState<SpotlightStyles>({
    left: 0,
    top: 0,
    width: 0,
    height: 0,
  });

  // Calculate position for the tour step tooltip
  const [tooltipPosition, setTooltipPosition] = useState({
    left: 0,
    top: 0,
    position: "bottom",
  });

  useEffect(() => {
    if (!isOpen || steps.length === 0) return;

    const updatePosition = () => {
      const currentStepData = steps[currentStep];
      const targetElement = document.querySelector(currentStepData.target);

      if (!targetElement) return;

      const rect = targetElement.getBoundingClientRect();

      // Add some padding to the spotlight
      const padding = 10;
      const spotlightStyles = {
        left: rect.left - padding,
        top: rect.top - padding,
        width: rect.width + padding * 2,
        height: rect.height + padding * 2,
      };

      setSpotlight(spotlightStyles);

      // Calculate tooltip position based on the step's position preference or available space
      const position =
        currentStepData.position || calculateOptimalPosition(rect);

      let tooltipLeft, tooltipTop;

      // Position the tooltip based on the specified position
      switch (position) {
        case "top":
          tooltipLeft = rect.left + rect.width / 2;
          tooltipTop = rect.top - 30; // Reduced distance
          break;
        case "bottom":
          tooltipLeft = rect.left + rect.width / 2;
          tooltipTop = rect.bottom + 10; // Reduced distance
          break;
        case "left":
          tooltipLeft = rect.left - 40; // Reduced distance
          tooltipTop = rect.top + rect.height / 2;
          break;
        case "right":
          tooltipLeft = rect.right + 40; // Reduced distance
          tooltipTop = rect.top + rect.height / 2;
          break;
        default:
          tooltipLeft = rect.left + rect.width / 2;
          tooltipTop = rect.bottom + 10; // Reduced distance
      }

      setTooltipPosition({
        left: tooltipLeft,
        top: tooltipTop,
        position,
      });
    };

    // Calculate optimal tooltip position based on available space
    const calculateOptimalPosition = (rect: DOMRect) => {
      const viewportHeight = window.innerHeight;
      const viewportWidth = window.innerWidth;

      // Determine if there's more space above or below the element
      const spaceAbove = rect.top;
      const spaceBelow = viewportHeight - rect.bottom;
      const spaceLeft = rect.left;
      const spaceRight = viewportWidth - rect.right;

      // Find the direction with the most space
      const maxSpace = Math.max(spaceAbove, spaceBelow, spaceLeft, spaceRight);

      if (maxSpace === spaceBelow) return "bottom";
      if (maxSpace === spaceAbove) return "top";
      if (maxSpace === spaceLeft) return "left";
      return "right";
    };

    // Update position initially and when window is resized
    updatePosition();
    window.addEventListener("resize", updatePosition);

    return () => {
      window.removeEventListener("resize", updatePosition);
    };
  }, [isOpen, currentStep, steps]);

  if (!isOpen || steps.length === 0) return null;

  // Create the tooltip class based on position
  const getTooltipClass = () => {
    switch (tooltipPosition.position) {
      case "top":
        return "translate-y-[-105%] -translate-x-1/2";
      case "bottom":
        return "-translate-x-1/2 translate-y-[5px]";
      case "left":
        return "translate-y-[-50%] -translate-x-[105%]";
      case "right":
        return "translate-y-[-50%] translate-x-[5px]";
      default:
        return "-translate-x-1/2 translate-y-[5px]";
    }
  };

  const tooltipClass = getTooltipClass();

  // Return null if we're between transitions or no valid step
  if (currentStep < 0 || currentStep >= steps.length) return null;

  const currentStepData = steps[currentStep];

  return (
    <Portal>
      <div className="fixed inset-0 z-[99999]" data-tour-overlay>
        <div
          className="absolute inset-0 overflow-hidden pointer-events-auto"
          onClick={(e) => {
            // Prevent clicks from reaching the dialog
            e.preventDefault();
            e.stopPropagation();
          }}
        >
          {/* Semi-transparent backdrop */}
          <div className="absolute inset-0 flex flex-col pointer-events-auto">
            {/* Top backdrop */}
            <div
              className="w-full bg-black/40"
              style={{ height: `${Math.max(0, spotlight.top)}px` }}
              onClick={(e) => {
                e.stopPropagation();
                closeTour();
              }}
            />

            {/* Middle row with spotlight */}
            <div className="flex" style={{ height: `${spotlight.height}px` }}>
              {/* Left of spotlight */}
              <div
                className="bg-black/40"
                style={{ width: `${Math.max(0, spotlight.left)}px` }}
                onClick={(e) => {
                  e.stopPropagation();
                  closeTour();
                }}
              />

              {/* Spotlight area - transparent */}
              <div
                className="relative"
                style={{ width: `${spotlight.width}px` }}
              >
                {/* Spotlight border */}
                <div className="absolute inset-0 rounded-lg pointer-events-none" />
              </div>

              {/* Right of spotlight */}
              <div
                className="bg-black/40 flex-1"
                onClick={(e) => {
                  e.stopPropagation();
                  closeTour();
                }}
              />
            </div>

            {/* Bottom backdrop */}
            <div
              className="w-full bg-black/40 flex-1"
              onClick={(e) => {
                e.stopPropagation();
                closeTour();
              }}
            />
          </div>

          {/* Tooltip */}
          <div
            className={cn(
              "fixed p-4 bg-background border border-border rounded-md shadow-xl z-[100001] w-80 pointer-events-auto",
              tooltipClass,
            )}
            style={{
              left: `${tooltipPosition.left}px`,
              top: `${tooltipPosition.top}px`,
            }}
            onClick={(e) => {
              e.stopPropagation();
            }}
          >
            <div className="flex justify-between items-center mb-2">
              <h3 className="font-semibold text-lg">{currentStepData.title}</h3>
              <Button
                variant="ghost"
                size="sm"
                type="button"
                className="h-6 w-6 p-0"
                onClick={(e) => {
                  e.stopPropagation();
                  closeTour();
                }}
              >
                <Icon icon={faXmark} className="size-4" />
              </Button>
            </div>

            <div className="text-sm text-muted-foreground mb-4">
              {currentStepData.content}
            </div>

            <div className="flex justify-between items-center">
              <div className="text-xs text-muted-foreground">
                Step {currentStep + 1} of {steps.length}
              </div>

              <div className="flex gap-2">
                {currentStep > 0 && (
                  <Button
                    variant="outline"
                    size="sm"
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      prevStep();
                    }}
                  >
                    <Icon icon={faArrowLeft} className="size-3" />
                    Previous
                  </Button>
                )}

                <Button
                  variant="default"
                  size="sm"
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    nextStep();
                  }}
                >
                  {currentStep < steps.length - 1 ? (
                    <>
                      Next
                      <Icon icon={faArrowRight} className="size-3" />
                    </>
                  ) : (
                    "Finish"
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Portal>
  );
}
