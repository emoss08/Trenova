import { useState } from 'react';

// material-ui
import { Box, Button, Step, Stepper, StepContent, StepLabel, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

const steps = [
  {
    label: 'Select campaign settings',
    description: `For each ad campaign that you create, you can control how much
              you're willing to spend on clicks and conversions, which networks
              and geographical locations you want your ads to show on, and more.`
  },
  {
    label: 'Create an ad group',
    description: 'An ad group contains one or more ads which target a shared set of keywords.'
  },
  {
    label: 'Create an ad',
    description: `Try out different ad text to see what brings in the most customers,
              and learn how to enhance your ads using features like ad extensions.
              If you run into any problems with your ads, find out how to tell if
              they're running and how to resolve approval issues.`
  }
];

// ==============================|| STEPPER - VERTICAL ||============================== //

export default function VerticalLinearStepper() {
  const [activeStep, setActiveStep] = useState(0);

  const handleNext = () => setActiveStep((prevActiveStep) => prevActiveStep + 1);
  const handleBack = () => setActiveStep((prevActiveStep) => prevActiveStep - 1);
  const handleReset = () => setActiveStep(0);

  const verticalStepperCodeString = `// VerticalLinearStepper.tsx
<Stepper activeStep={activeStep} orientation="vertical">
  {steps.map((step, index) => (
    <Step key={step.label}>
      <StepLabel optional={index === 2 ? <Typography variant="caption">Last step</Typography> : null}>{step.label}</StepLabel>
      <StepContent>
        <Typography>{step.description}</Typography>
        <Box sx={{ mb: 2 }}>
          <div>
            <Button
              variant="contained"
              onClick={handleNext}
              sx={{ mt: 1, mr: 1 }}
              color={index === steps.length - 1 ? 'success' : 'primary'}
            >
              {index === steps.length - 1 ? 'Finish' : 'Continue'}
            </Button>
            <Button disabled={index === 0} onClick={handleBack} sx={{ mt: 1, mr: 1 }}>
              Back
            </Button>
          </div>
        </Box>
      </StepContent>
    </Step>
  ))}
</Stepper>
{activeStep === steps.length && (
  <Box sx={{ pt: 2 }}>
    <Typography sx={{ color: 'success.main' }}>All steps completed - you&apos;re finished</Typography>
    <Button size="small" variant="contained" color="error" onClick={handleReset} sx={{ mt: 2, mr: 1 }}>
      Reset
    </Button>
  </Box>
)}`;

  return (
    <MainCard title="Vertical" codeString={verticalStepperCodeString}>
      <Stepper activeStep={activeStep} orientation="vertical">
        {steps.map((step, index) => (
          <Step key={step.label}>
            <StepLabel optional={index === 2 ? <Typography variant="caption">Last step</Typography> : null}>{step.label}</StepLabel>
            <StepContent>
              <Typography>{step.description}</Typography>
              <Box sx={{ mb: 2 }}>
                <div>
                  <Button
                    variant="contained"
                    onClick={handleNext}
                    sx={{ mt: 1, mr: 1 }}
                    color={index === steps.length - 1 ? 'success' : 'primary'}
                  >
                    {index === steps.length - 1 ? 'Finish' : 'Continue'}
                  </Button>
                  <Button disabled={index === 0} onClick={handleBack} sx={{ mt: 1, mr: 1 }}>
                    Back
                  </Button>
                </div>
              </Box>
            </StepContent>
          </Step>
        ))}
      </Stepper>
      {activeStep === steps.length && (
        <Box sx={{ pt: 2 }}>
          <Typography sx={{ color: 'success.main' }}>All steps completed - you&apos;re finished</Typography>
          <Button size="small" variant="contained" color="error" onClick={handleReset} sx={{ mt: 2, mr: 1 }}>
            Reset
          </Button>
        </Box>
      )}
    </MainCard>
  );
}
