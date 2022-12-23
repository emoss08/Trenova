// material-ui
import { Autocomplete, TextField } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import data from 'data/movies';

// ==============================|| AUTOCOMPLETE - GROUPED ||============================== //

export default function GroupedAutocomplete() {
  const options = data.map((option) => {
    const firstLetter = option.label[0].toUpperCase();
    return {
      firstLetter: /[0-9]/.test(firstLetter) ? '0-9' : firstLetter,
      ...option
    };
  });

  const groupAutocompleteCodeString = `<Autocomplete
  id="grouped-demo"
  fullWidth
  options={options.sort((a, b) => -b.firstLetter.localeCompare(a.firstLetter))}
  groupBy={(option) => option.firstLetter}
  getOptionLabel={(option) => option.label}
  renderInput={(params) => <TextField {...params} label="With categories" />}
/>`;

  return (
    <MainCard title="Grouped" codeString={groupAutocompleteCodeString}>
      <Autocomplete
        id="grouped-demo"
        fullWidth
        options={options.sort((a, b) => -b.firstLetter.localeCompare(a.firstLetter))}
        groupBy={(option) => option.firstLetter}
        getOptionLabel={(option) => option.label}
        renderInput={(params) => <TextField {...params} label="With categories" />}
      />
    </MainCard>
  );
}
