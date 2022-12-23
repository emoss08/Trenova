import { useState } from 'react';

// material-ui
import TextField from '@mui/material/TextField';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { DatePicker, LocalizationProvider } from '@mui/x-date-pickers';

// project import
import MainCard from 'components/MainCard';

// ==============================|| DATE PICKER - HELPER TEXT ||============================== //

export default function HelperText() {
  const [value, setValue] = useState<Date | null>(null);

  const helperDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <DatePicker
    label="Helper Text"
    value={value}
    onChange={(newValue) => {
      setValue(newValue);
    }}
    renderInput={(params) => <TextField {...params} helperText={params?.inputProps?.placeholder} />}
  />
</LocalizationProvider>`;

  return (
    <MainCard title="Helper Text" codeString={helperDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <DatePicker
          label="Helper Text"
          value={value}
          onChange={(newValue) => {
            setValue(newValue);
          }}
          renderInput={(params) => <TextField {...params} helperText={params?.inputProps?.placeholder} />}
        />
      </LocalizationProvider>
    </MainCard>
  );
}
