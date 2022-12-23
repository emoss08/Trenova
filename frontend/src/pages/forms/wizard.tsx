// material-ui
import { Grid } from '@mui/material';

// project imports
import BasicWizard from 'sections/forms/wizard/basic-wizard';
import ValidationWizard from 'sections/forms/wizard/validation-wizard';

// ==============================|| FORMS WIZARD ||============================== //

const FormsWizard = () => (
  <Grid container spacing={2.5} justifyContent="center">
    <Grid item xs={12} md={6} lg={7}>
      <BasicWizard />
    </Grid>
    <Grid item xs={12} md={6} lg={7}>
      <ValidationWizard />
    </Grid>
  </Grid>
);

export default FormsWizard;
