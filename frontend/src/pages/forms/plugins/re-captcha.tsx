// material ui
import { Grid } from '@mui/material';

// third-party
import ReCAPTCHA from 'react-google-recaptcha';

// project imports
import MainCard from 'components/MainCard';

// ==============================|| PLUGIN - RECAPTCHA ||============================== //

const RecaptchaPage = () => {
  const handleOnChange = () => {};
  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={12} lg={6}>
        <MainCard title="ReCaptcha Example">
          <ReCAPTCHA sitekey="6LdzqbcaAAAAALrGEZWQHIHUhzJZc8O-KSTdTTh_" onChange={handleOnChange} />
        </MainCard>
      </Grid>
    </Grid>
  );
};

export default RecaptchaPage;
