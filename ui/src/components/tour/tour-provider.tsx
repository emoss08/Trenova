import { createContext, ReactNode, useContext, useState } from "react";

interface TourStep {
  target: string; // CSS selector for the target element
  title: string;
  content: string;
  position?: "top" | "right" | "bottom" | "left";
  action?: () => void; // Function to execute when the step is shown
  actionOnNext?: () => void; // Function to execute when moving to the next step
  actionOnPrev?: () => void; // Function to execute when moving to the previous step
}

interface TourContextProps {
  isOpen: boolean;
  currentStep: number;
  steps: TourStep[];
  openTour: (steps: TourStep[], cleanup?: () => void) => void;
  closeTour: () => void;
  nextStep: () => void;
  prevStep: () => void;
  goToStep: (step: number) => void;
}

const TourContext = createContext<TourContextProps | undefined>(undefined);

export function useTour() {
  const context = useContext(TourContext);
  if (!context) {
    throw new Error("useTour must be used within a TourProvider");
  }
  return context;
}

interface TourProviderProps {
  children: ReactNode;
}

export function TourProvider({ children }: TourProviderProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [steps, setSteps] = useState<TourStep[]>([]);
  const [cleanupFn, setCleanupFn] = useState<(() => void) | undefined>(undefined);

  const openTour = (newSteps: TourStep[], cleanup?: () => void) => {
    setSteps(newSteps);
    setCurrentStep(0);
    setIsOpen(true);
    if (cleanup) {
      setCleanupFn(() => cleanup);
    }
    
    // Execute the action for the first step if it exists
    if (newSteps.length > 0 && newSteps[0].action) {
      setTimeout(() => {
        newSteps[0].action?.();
      }, 300); // Small delay to ensure the tour is rendered
    }
  };

  const closeTour = () => {
    setIsOpen(false);
    // Run cleanup function if it exists
    if (cleanupFn) {
      setTimeout(() => {
        cleanupFn();
        setCleanupFn(undefined);
      }, 300); // Small delay to ensure the tour is fully closed
    }
  };

  const nextStep = () => {
    if (currentStep < steps.length - 1) {
      // Execute the actionOnNext for the current step if it exists
      if (steps[currentStep].actionOnNext) {
        steps[currentStep].actionOnNext?.();
      }
      
      const nextStepIndex = currentStep + 1;
      setCurrentStep(nextStepIndex);
      
      // Execute the action for the next step if it exists
      if (steps[nextStepIndex].action) {
        setTimeout(() => {
          steps[nextStepIndex].action?.();
        }, 300); // Small delay to ensure the step transition is complete
      }
    } else {
      closeTour();
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      // Execute the actionOnPrev for the current step if it exists
      if (steps[currentStep].actionOnPrev) {
        steps[currentStep].actionOnPrev?.();
      }
      
      const prevStepIndex = currentStep - 1;
      setCurrentStep(prevStepIndex);
      
      // Execute the action for the previous step if it exists
      if (steps[prevStepIndex].action) {
        setTimeout(() => {
          steps[prevStepIndex].action?.();
        }, 300); // Small delay to ensure the step transition is complete
      }
    }
  };

  const goToStep = (step: number) => {
    if (step >= 0 && step < steps.length) {
      setCurrentStep(step);
      
      // Execute the action for the destination step if it exists
      if (steps[step].action) {
        setTimeout(() => {
          steps[step].action?.();
        }, 300); // Small delay to ensure the step transition is complete
      }
    }
  };

  return (
    <TourContext.Provider
      value={{
        isOpen,
        currentStep,
        steps,
        openTour,
        closeTour,
        nextStep,
        prevStep,
        goToStep,
      }}
    >
      {children}
    </TourContext.Provider>
  );
}
