import { useState } from 'react';

// material-ui
import { Stack, TextField } from '@mui/material';

import { LocalizationProvider, DateTimePicker, DesktopDatePicker, MobileDatePicker, TimePicker } from '@mui/x-date-pickers';

import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - BASIC ||============================== //

export default function BasicDateTimePickers() {
  const [value, setValue] = useState<Date | null>(new Date('2014-08-18T21:11:54'));

  const handleChange = (newValue: Date | null) => {
    setValue(newValue);
  };

  const basicDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <Stack spacing={3}>
    <DesktopDatePicker
      label="Date Desktop"
      inputFormat="MM/dd/yyyy"
      value={value}
      onChange={handleChange}
      renderInput={(params) => <TextField {...params} />}
    />
    <MobileDatePicker
      label="Date Mobile"
      inputFormat="MM/dd/yyyy"
      value={value}
      onChange={handleChange}
      renderInput={(params) => <TextField {...params} />}
    />
    <TimePicker label="Time" value={value} onChange={handleChange} renderInput={(params) => <TextField {...params} />} />
    <DateTimePicker
      label="Date & Time Picker"
      value={value}
      onChange={handleChange}
      renderInput={(params) => <TextField {...params} />}
    />
  </Stack>
</LocalizationProvider>`;

  return (
    <MainCard title="Basic Picker" codeString={basicDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Stack spacing={3}>
          <DesktopDatePicker
            label="Date Desktop"
            inputFormat="MM/dd/yyyy"
            value={value}
            onChange={handleChange}
            renderInput={(params) => <TextField {...params} />}
          />
          <MobileDatePicker
            label="Date Mobile"
            inputFormat="MM/dd/yyyy"
            value={value}
            onChange={handleChange}
            renderInput={(params) => <TextField {...params} />}
          />
          <TimePicker label="Time" value={value} onChange={handleChange} renderInput={(params) => <TextField {...params} />} />
          <DateTimePicker
            label="Date & Time Picker"
            value={value}
            onChange={handleChange}
            renderInput={(params) => <TextField {...params} />}
          />
        </Stack>
      </LocalizationProvider>
    </MainCard>
  );
}
