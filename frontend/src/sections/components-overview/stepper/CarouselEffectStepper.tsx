import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, MobileStepper, Paper, Typography } from '@mui/material';

// third-party
import SwipeableViews from 'react-swipeable-views';
import { autoPlay } from 'react-swipeable-views-utils';

// project import
import MainCard from 'components/MainCard';

// assets
import { RightOutlined, LeftOutlined } from '@ant-design/icons';

const AutoPlaySwipeableViews = autoPlay(SwipeableViews);

const images = [
  {
    label: 'San Francisco – Oakland Bay Bridge, United States',
    imgPath: 'https://images.unsplash.com/photo-1537944434965-cf4679d1a598?auto=format&fit=crop&w=400&h=250&q=60'
  },
  {
    label: 'Bird',
    imgPath: 'https://images.unsplash.com/photo-1538032746644-0212e812a9e7?auto=format&fit=crop&w=400&h=250&q=60'
  },
  {
    label: 'Bali, Indonesia',
    imgPath: 'https://images.unsplash.com/photo-1537996194471-e657df975ab4?auto=format&fit=crop&w=400&h=250&q=80'
  },
  {
    label: 'Goč, Serbia',
    imgPath: 'https://images.unsplash.com/photo-1512341689857-198e7e2f3ca8?auto=format&fit=crop&w=400&h=250&q=60'
  }
];

// ==============================|| STEPPER - CAROUSEL EFFECT ||============================== //

function CarouselEffectStepper() {
  const theme = useTheme();
  const [activeStep, setActiveStep] = useState(0);
  const maxSteps = images.length;

  const handleNext = () => setActiveStep((prevActiveStep) => prevActiveStep + 1);
  const handleBack = () => setActiveStep((prevActiveStep) => prevActiveStep - 1);
  const handleStepChange = (step: number) => setActiveStep(step);

  const carouselStepperCodeString = `// CarouselEffectStepper.tsx
<Paper
  square
  elevation={0}
  sx={{
    display: 'flex',
    alignItems: 'center',
    height: 50,
    pl: 2,
    bgcolor: 'background.paper'
  }}
>
  <Typography>{images[activeStep].label}</Typography>
</Paper>
<AutoPlaySwipeableViews
  axis={theme.direction === 'rtl' ? 'x-reverse' : 'x'}
  index={activeStep}
  onChangeIndex={handleStepChange}
  enableMouseEvents
>
  {images.map((step, index) => (
    <Box key={step.label}>
      {Math.abs(activeStep - index) <= 2 ? (
        <Box
          component="img"
          sx={{
            height: 255,
            display: 'block',
            overflow: 'hidden',
            width: '100%'
          }}
          src={step.imgPath}
          alt={step.label}
        />
      ) : null}
    </Box>
  ))}
</AutoPlaySwipeableViews>
<MobileStepper
  sx={{ bgcolor: 'background.paper', '& .anticon': { fontSize: '0.625rem' } }}
  steps={maxSteps}
  position="static"
  activeStep={activeStep}
  nextButton={
    <Button
      size="small"
      onClick={handleNext}
      disabled={activeStep === maxSteps - 1}
      endIcon={theme.direction === 'rtl' ? <LeftOutlined /> : <RightOutlined />}
    >
      Next
    </Button>
  }
  backButton={
    <Button
      size="small"
      onClick={handleBack}
      disabled={activeStep === 0}
      startIcon={theme.direction === 'rtl' ? <RightOutlined /> : <LeftOutlined />}
    >
      Back
    </Button>
  }
/>`;

  return (
    <MainCard sx={{ flexGrow: 1 }} content={false} codeString={carouselStepperCodeString}>
      <Paper
        square
        elevation={0}
        sx={{
          display: 'flex',
          alignItems: 'center',
          height: 50,
          pl: 2,
          bgcolor: 'background.paper'
        }}
      >
        <Typography>{images[activeStep].label}</Typography>
      </Paper>
      <AutoPlaySwipeableViews
        axis={theme.direction === 'rtl' ? 'x-reverse' : 'x'}
        index={activeStep}
        onChangeIndex={handleStepChange}
        enableMouseEvents
      >
        {images.map((step, index) => (
          <Box key={step.label}>
            {Math.abs(activeStep - index) <= 2 ? (
              <Box
                component="img"
                sx={{
                  height: 255,
                  display: 'block',
                  overflow: 'hidden',
                  width: '100%'
                }}
                src={step.imgPath}
                alt={step.label}
              />
            ) : null}
          </Box>
        ))}
      </AutoPlaySwipeableViews>
      <MobileStepper
        sx={{ bgcolor: 'background.paper', '& .anticon': { fontSize: '0.625rem' } }}
        steps={maxSteps}
        position="static"
        activeStep={activeStep}
        nextButton={
          <Button
            size="small"
            onClick={handleNext}
            disabled={activeStep === maxSteps - 1}
            endIcon={theme.direction === 'rtl' ? <LeftOutlined /> : <RightOutlined />}
          >
            Next
          </Button>
        }
        backButton={
          <Button
            size="small"
            onClick={handleBack}
            disabled={activeStep === 0}
            startIcon={theme.direction === 'rtl' ? <RightOutlined /> : <LeftOutlined />}
          >
            Back
          </Button>
        }
      />
    </MainCard>
  );
}

export default CarouselEffectStepper;
