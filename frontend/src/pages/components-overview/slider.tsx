import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Grid, Stack, Slider } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import { AudioMutedOutlined, SoundOutlined } from '@ant-design/icons';

function valuetext(value: number) {
  return `${value}°C`;
}

function valueLabelFormat(value: number) {
  return marks.findIndex((mark) => mark.value === value) + 1;
}

const minDistance = 10;

const marks = [
  {
    value: 0,
    label: '0°C'
  },
  {
    value: 20,
    label: '20°C'
  },
  {
    value: 37,
    label: '37°C'
  },
  {
    value: 100,
    label: '100°C'
  }
];

// ==============================|| COMPONENTS - SLIDER ||============================== //

const ComponentSlider = () => {
  const theme = useTheme();

  const [volume, setVolume] = useState<number>(55);
  const handleVolumeChange = (event: Event, newVolume: number | number[]) => {
    setVolume(newVolume as number);
  };

  const [range, setRange] = useState<number[]>([20, 37]);
  const handleRangeChange = (event: Event, newRange: number | number[]) => {
    setRange(newRange as number[]);
  };

  const [value1, setValue1] = useState<number[]>([20, 55]);
  const handleChange1 = (event: Event, newValue: number | number[], activeThumb: number) => {
    if (!Array.isArray(newValue)) {
      return;
    }

    if (activeThumb === 0) {
      setValue1([Math.min(newValue[0], value1[1] - minDistance), value1[1]]);
    } else {
      setValue1([value1[0], Math.max(newValue[1], value1[0] + minDistance)]);
    }
  };

  const [value2, setValue2] = useState<number[]>([35, 76]);
  const handleChange2 = (event: Event, newValue: number | number[], activeThumb: number) => {
    if (!Array.isArray(newValue)) {
      return;
    }

    if (newValue[1] - newValue[0] < minDistance) {
      if (activeThumb === 0) {
        const clamped = Math.min(newValue[0], 100 - minDistance);
        setValue2([clamped, clamped + minDistance]);
      } else {
        const clamped = Math.max(newValue[1], minDistance);
        setValue2([clamped - minDistance, clamped]);
      }
    } else {
      setValue2(newValue as number[]);
    }
  };

  const basicSliderCodeString = `<Slider defaultValue={35} />`;

  const iconsSliderCodeString = `<Stack spacing={2} direction="row" sx={{ mb: 1 }} alignItems="center">
  <AudioMutedOutlined style={{ color: volume <= 25 ? 'inherit' : theme.palette.text.secondary }} />
  <Slider aria-label="Volume" value={volume} onChange={handleVolumeChange} />
  <SoundOutlined style={{ color: volume > 25 ? 'inherit' : theme.palette.text.secondary }} />
</Stack>`;

  const rangesSliderCodeString = `<Slider
  getAriaLabel={() => 'Temperature range'}
  value={range}
  onChange={handleRangeChange}
  valueLabelDisplay="auto"
  getAriaValueText={valuetext}
/>`;

  const labelSliderCodeString = `<Slider
  sx={{ mt: 2.5 }}
  aria-label="Always visible"
  defaultValue={80}
  getAriaValueText={valuetext}
  step={10}
  valueLabelDisplay="on"
/>`;

  const verticalSliderCodeString = `<Stack sx={{ height: 300 }} spacing={1} direction="row">
  <Slider aria-label="Temperature" orientation="vertical" getAriaValueText={valuetext} defaultValue={30} />
  <Slider aria-label="Temperature" orientation="vertical" defaultValue={30} disabled />
  <Slider
    getAriaLabel={() => 'Temperature'}
    orientation="vertical"
    getAriaValueText={valuetext}
    defaultValue={[20, 37]}
    marks={marks}
    color="warning"
  />
</Stack>`;

  const disabledSliderCodeString = `<Slider defaultValue={50} disabled />`;

  const sizeSliderCodeString = `<Slider size="small" defaultValue={70} aria-label="Small" valueLabelDisplay="auto" />
<Slider defaultValue={50} aria-label="Default" valueLabelDisplay="auto" />`;

  const discreteSliderCodeString = `<Slider
  aria-label="Temperature"
  defaultValue={60}
  getAriaValueText={valuetext}
  valueLabelDisplay="auto"
  step={10}
  marks
  min={10}
  max={110}
/>`;

  const restrictedSliderCodeString = `<Slider
  aria-label="Restricted values"
  defaultValue={20}
  valueLabelFormat={valueLabelFormat}
  getAriaValueText={valuetext}
  step={null}
  valueLabelDisplay="auto"
  marks={marks}
/>`;

  const minSliderCodeString = `<Slider
  getAriaLabel={() => 'Minimum distance'}
  value={value1}
  onChange={handleChange1}
  valueLabelDisplay="auto"
  getAriaValueText={valuetext}
  disableSwap
/>
<Slider
  getAriaLabel={() => 'Minimum distance shift'}
  value={value2}
  onChange={handleChange2}
  valueLabelDisplay="auto"
  getAriaValueText={valuetext}
  disableSwap
/>`;

  const colorsSliderCodeString = `<Slider defaultValue={65} />
<Slider defaultValue={50} color="secondary" />
<Slider defaultValue={95} color="success" />
<Slider defaultValue={30} color="warning" />
<Slider defaultValue={85} color="info" />
<Slider defaultValue={5} color="error" />`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Slider"
        caption="Sliders allow users to make selections from a range of values."
        directory="src/pages/components-overview/slider"
        link="https://mui.com/material-ui/react-slider/"
      />
      <ComponentWrapper>
        <Grid container spacing={2.5}>
          <Grid item xs={12} sm={6}>
            <Stack spacing={2.5}>
              <MainCard title="Basic" codeHighlight codeString={basicSliderCodeString}>
                <Slider defaultValue={35} />
              </MainCard>
              <MainCard title="With Icons" codeString={iconsSliderCodeString}>
                <Stack spacing={2} direction="row" sx={{ mb: 1 }} alignItems="center">
                  <AudioMutedOutlined style={{ color: volume <= 25 ? 'inherit' : theme.palette.text.secondary }} />
                  <Slider aria-label="Volume" value={volume} onChange={handleVolumeChange} />
                  <SoundOutlined style={{ color: volume > 25 ? 'inherit' : theme.palette.text.secondary }} />
                </Stack>
              </MainCard>
              <MainCard title="Range" codeString={rangesSliderCodeString}>
                <Slider
                  getAriaLabel={() => 'Temperature range'}
                  value={range}
                  onChange={handleRangeChange}
                  valueLabelDisplay="auto"
                  getAriaValueText={valuetext}
                />
              </MainCard>
              <MainCard title="With Label" codeString={labelSliderCodeString}>
                <Slider
                  sx={{ mt: 2.5 }}
                  aria-label="Always visible"
                  defaultValue={80}
                  getAriaValueText={valuetext}
                  step={10}
                  valueLabelDisplay="on"
                />
              </MainCard>
              <MainCard title="Vertical" codeString={verticalSliderCodeString}>
                <Stack sx={{ height: 300 }} spacing={1} direction="row">
                  <Slider aria-label="Temperature" orientation="vertical" getAriaValueText={valuetext} defaultValue={30} />
                  <Slider aria-label="Temperature" orientation="vertical" defaultValue={30} disabled />
                  <Slider
                    getAriaLabel={() => 'Temperature'}
                    orientation="vertical"
                    getAriaValueText={valuetext}
                    defaultValue={[20, 37]}
                    marks={marks}
                    color="warning"
                  />
                </Stack>
              </MainCard>
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={2.5}>
              <MainCard title="Disabled" codeString={disabledSliderCodeString}>
                <Slider defaultValue={50} disabled />
              </MainCard>
              <MainCard title="Sizes" codeString={sizeSliderCodeString}>
                <Slider size="small" defaultValue={70} aria-label="Small" valueLabelDisplay="auto" />
                <Slider defaultValue={50} aria-label="Default" valueLabelDisplay="auto" />
              </MainCard>
              <MainCard title="Discrete" codeString={discreteSliderCodeString}>
                <Slider
                  aria-label="Temperature"
                  defaultValue={60}
                  getAriaValueText={valuetext}
                  valueLabelDisplay="auto"
                  step={10}
                  marks
                  min={10}
                  max={110}
                />
              </MainCard>
              <MainCard title="Restricted values" codeString={restrictedSliderCodeString}>
                <Slider
                  aria-label="Restricted values"
                  defaultValue={20}
                  valueLabelFormat={valueLabelFormat}
                  getAriaValueText={valuetext}
                  step={null}
                  valueLabelDisplay="auto"
                  marks={marks}
                />
              </MainCard>
              <MainCard title="Minimum distance" codeString={minSliderCodeString}>
                <Slider
                  getAriaLabel={() => 'Minimum distance'}
                  value={value1}
                  onChange={handleChange1}
                  valueLabelDisplay="auto"
                  getAriaValueText={valuetext}
                  disableSwap
                />
                <Slider
                  getAriaLabel={() => 'Minimum distance shift'}
                  value={value2}
                  onChange={handleChange2}
                  valueLabelDisplay="auto"
                  getAriaValueText={valuetext}
                  disableSwap
                />
              </MainCard>
              <MainCard title="Colors" codeString={colorsSliderCodeString}>
                <Slider defaultValue={65} />
                <Slider defaultValue={50} color="secondary" />
                <Slider defaultValue={95} color="success" />
                <Slider defaultValue={30} color="warning" />
                <Slider defaultValue={85} color="info" />
                <Slider defaultValue={5} color="error" />
              </MainCard>
            </Stack>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentSlider;
