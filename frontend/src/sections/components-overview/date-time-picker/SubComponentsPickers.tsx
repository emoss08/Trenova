import { useState } from 'react';

// material-ui
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { Box, Stack } from '@mui/material';
import { CalendarPicker, LocalizationProvider, MonthPicker, YearPicker } from '@mui/x-date-pickers';

// project import
import MainCard from 'components/MainCard';

const minDate = new Date('2020-01-01T00:00:00.000');
const maxDate = new Date('2034-01-01T00:00:00.000');

// ==============================|| DATE PICKER - SUB COMPONENT ||============================== //

export default function SubComponentsPickers() {
  const [date, setDate] = useState<Date | null>(new Date());

  const subDatepickerCodeString = `<LocalizationProvider dateAdapter={AdapterDateFns}>
  <Stack spacing={3} justifyContent="center" alignItems="center">
    <Box sx={{ maxWidth: 320 }}>
      <YearPicker
        date={date}
        isDateDisabled={() => false}
        minDate={minDate}
        maxDate={maxDate}
        onChange={(newDate) => setDate(newDate)}
      />
    </Box>
    <Box sx={{ maxWidth: 320 }}>
      <MonthPicker date={date} minDate={minDate} maxDate={maxDate} onChange={(newDate) => setDate(newDate)} sx={{ m: 'auto' }} />
    </Box>
    <Box sx={{ maxWidth: 320 }}>
      <CalendarPicker date={date} onChange={(newDate) => setDate(newDate)} />
    </Box>
  </Stack>
</LocalizationProvider>`;

  return (
    <MainCard title="Sub Component" codeString={subDatepickerCodeString}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Stack spacing={3} justifyContent="center" alignItems="center">
          <Box sx={{ maxWidth: 320 }}>
            <YearPicker
              date={date}
              // @ts-ignore
              isDateDisabled={() => false}
              minDate={minDate}
              maxDate={maxDate}
              onChange={(newDate: Date) => setDate(newDate)}
            />
          </Box>
          <Box sx={{ maxWidth: 320 }}>
            <MonthPicker
              date={date}
              minDate={minDate}
              maxDate={maxDate}
              onChange={(newDate: Date) => setDate(newDate)}
              sx={{ m: 'auto' }}
            />
          </Box>
          <Box sx={{ maxWidth: 320 }}>
            {/* @ts-ignore */}
            <CalendarPicker date={date} onChange={(newDate: Date) => setDate(newDate)} />
          </Box>
        </Stack>
      </LocalizationProvider>
    </MainCard>
  );
}
