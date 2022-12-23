import { useState } from 'react';

// material-ui
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { TextField } from '@mui/material';
import { LocalizationProvider, StaticDatePicker } from '@mui/x-date-pickers';

// third-party
import isWeekend from 'date-fns/isWeekend';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - LANDSCAPE ||============================== //

export default function LandscapeDatePicker() {
  const [value, setValue] = useState<Date | null>(new Date());

  const landscapDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <StaticDatePicker<Date>
    orientation="landscape"
    openTo="day"
    value={value}
    shouldDisableDate={isWeekend}
    onChange={(newValue) => {
      setValue(newValue);
    }}
    renderInput={(params) => <TextField {...params} />}
  />
</LocalizationProvider>`;

  return (
    <MainCard title="Landscape" codeString={landscapDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <StaticDatePicker<Date>
          orientation="landscape"
          openTo="day"
          value={value}
          shouldDisableDate={isWeekend}
          onChange={(newValue) => {
            setValue(newValue);
          }}
          renderInput={(params) => <TextField {...params} />}
        />
      </LocalizationProvider>
    </MainCard>
  );
}
