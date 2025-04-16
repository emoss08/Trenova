import { createContext, ReactNode, useContext, useState } from "react";

interface TourStep {
  target: string; // CSS selector for the target element
  title: string;
  content: string;
  position?: "top" | "right" | "bottom" | "left";
}

interface TourContextProps {
  isOpen: boolean;
  currentStep: number;
  steps: TourStep[];
  openTour: (steps: TourStep[]) => void;
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

  const openTour = (newSteps: TourStep[]) => {
    setSteps(newSteps);
    setCurrentStep(0);
    setIsOpen(true);
  };

  const closeTour = () => {
    setIsOpen(false);
  };

  const nextStep = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep((prev) => prev + 1);
    } else {
      closeTour();
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep((prev) => prev - 1);
    }
  };

  const goToStep = (step: number) => {
    if (step >= 0 && step < steps.length) {
      setCurrentStep(step);
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
