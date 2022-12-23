import { useState } from 'react';

// material-ui
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { Box, Stack, TextField, Typography } from '@mui/material';
import { DatePicker, DateRange, DateRangePicker, DateTimePicker, LocalizationProvider, TimePicker } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - DISABLED ||============================== //

export default function DisabledPickers() {
  const [value, setValue] = useState<Date | null>(null);
  const [valueRange, setValueRange] = useState<DateRange<Date>>([null, null]);

  const disabledDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <Stack spacing={3}>
    <Typography variant="h6">Date Picker</Typography>
    <DatePicker
      label="disabled"
      disabled
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
    <DatePicker
      label="read-only"
      readOnly
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
    <Typography variant="h6">Date Range Picker</Typography>
    <DateRangePicker
      disabled
      startText="disabled start"
      endText="disabled end"
      value={valueRange}
      onChange={(newValue: any) => {
        setValueRange(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> to </Box>
          <TextField {...endProps} />
        </>
      )}
    />
    <DateRangePicker
      readOnly
      startText="read-only start"
      endText="read-only end"
      value={valueRange}
      onChange={(newValue: any) => {
        setValueRange(newValue);
      }}
      renderInput={(startProps: any, endProps: any) => (
        <>
          <TextField {...startProps} />
          <Box sx={{ mx: 2 }}> to </Box>
          <TextField {...endProps} />
        </>
      )}
    />
    <Typography variant="h6">Date Time Picker</Typography>
    <DateTimePicker
      label="disabled"
      disabled
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
    <DateTimePicker
      label="read-only"
      readOnly
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
    <Typography variant="h6">Time Picker</Typography>
    <TimePicker
      label="disabled"
      disabled
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
    <TimePicker
      label="read-only"
      readOnly
      value={value}
      onChange={(newValue: any) => {
        setValue(newValue);
      }}
      renderInput={(params: any) => <TextField {...params} />}
    />
  </Stack>
</LocalizationProvider>`;

  return (
    <MainCard title="Disabled Pickers" codeString={disabledDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Stack spacing={3}>
          <Typography variant="h6">Date Picker</Typography>
          <DatePicker
            label="disabled"
            disabled
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
          <DatePicker
            label="read-only"
            readOnly
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
          <Typography variant="h6">Date Range Picker</Typography>
          <DateRangePicker
            disabled
            startText="disabled start"
            endText="disabled end"
            value={valueRange}
            onChange={(newValue: any) => {
              setValueRange(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> to </Box>
                <TextField {...endProps} />
              </>
            )}
          />
          <DateRangePicker
            readOnly
            startText="read-only start"
            endText="read-only end"
            value={valueRange}
            onChange={(newValue: any) => {
              setValueRange(newValue);
            }}
            renderInput={(startProps: any, endProps: any) => (
              <>
                <TextField {...startProps} />
                <Box sx={{ mx: 2 }}> to </Box>
                <TextField {...endProps} />
              </>
            )}
          />
          <Typography variant="h6">Date Time Picker</Typography>
          <DateTimePicker
            label="disabled"
            disabled
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
          <DateTimePicker
            label="read-only"
            readOnly
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
          <Typography variant="h6">Time Picker</Typography>
          <TimePicker
            label="disabled"
            disabled
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
          <TimePicker
            label="read-only"
            readOnly
            value={value}
            onChange={(newValue: any) => {
              setValue(newValue);
            }}
            renderInput={(params: any) => <TextField {...params} />}
          />
        </Stack>
      </LocalizationProvider>
    </MainCard>
  );
}
