import { useState } from 'react';

// material-ui
import { Box, TextField, Typography } from '@mui/material';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { DateRangePicker, DateRange, LocalizationProvider } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - NO. OF CALENDERS ||============================== //

export default function CalendarsRangePicker() {
  const [value, setValue] = useState<DateRange<Date>>([null, null]);

  const calDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <div>
    <Typography sx={{ mt: 2, mb: 1 }}>1 calendar </Typography>
    <DateRangePicker
      calendars={1}
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> to </Box>
          <TextField {...endProps} />
        </>
      )}
    />
    <Typography sx={{ mt: 2, mb: 1 }}>2 calendars</Typography>
    <DateRangePicker
      calendars={2}
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> to </Box>
          <TextField {...endProps} />
        </>
      )}
    />
    <Typography sx={{ mt: 2, mb: 1 }}>3 calendars</Typography>
    <DateRangePicker
      calendars={3}
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> to </Box>
          <TextField {...endProps} />
        </>
      )}
    />
  </div>
</LocalizationProvider>`;

  return (
    <MainCard title="Calendars Range Picker" codeString={calDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <div>
          <Typography sx={{ mt: 2, mb: 1 }}>1 calendar </Typography>
          <DateRangePicker
            calendars={1}
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> to </Box>
                <TextField {...endProps} />
              </>
            )}
          />
          <Typography sx={{ mt: 2, mb: 1 }}>2 calendars</Typography>
          <DateRangePicker
            calendars={2}
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> to </Box>
                <TextField {...endProps} />
              </>
            )}
          />
          <Typography sx={{ mt: 2, mb: 1 }}>3 calendars</Typography>
          <DateRangePicker
            calendars={3}
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> to </Box>
                <TextField {...endProps} />
              </>
            )}
          />
        </div>
      </LocalizationProvider>
    </MainCard>
  );
}
