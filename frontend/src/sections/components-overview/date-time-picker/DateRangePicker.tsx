import { useState } from 'react';

// material-ui
import { Box, Stack, TextField } from '@mui/material';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { DateRange, DesktopDateRangePicker, LocalizationProvider, MobileDateRangePicker } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - DATE RANGE ||============================== //

export default function DateRangePicker() {
  const [value, setValue] = useState<DateRange<Date>>([null, null]);

  const rangeDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <Stack spacing={3}>
    <MobileDateRangePicker
      startText="Mobile Start"
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> To </Box>
          <TextField {...endProps} />
        </>
      )}
    />
    <DesktopDateRangePicker
      startText="Desktop Start"
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> To </Box>
          <TextField {...endProps} />
        </>
      )}
    />
  </Stack>
</LocalizationProvider>`;

  return (
    <MainCard title="Date Range Picker" codeString={rangeDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Stack spacing={3}>
          <MobileDateRangePicker
            startText="Mobile Start"
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> To </Box>
                <TextField {...endProps} />
              </>
            )}
          />
          <DesktopDateRangePicker
            startText="Desktop Start"
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> To </Box>
                <TextField {...endProps} />
              </>
            )}
          />
        </Stack>
      </LocalizationProvider>
    </MainCard>
  );
}
