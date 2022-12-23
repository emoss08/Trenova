import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, Checkbox, Grid, MenuItem, Stack, TextField, Typography } from '@mui/material';

// select project-budget
const currencies = [
  {
    value: '1',
    label: 'Below $1000'
  },
  {
    value: '2',
    label: '$1000 - $5000'
  },
  {
    value: '3',
    label: 'Not specified'
  }
];

// select company-size
const sizes = [
  {
    value: '1',
    label: '1 - 5'
  },
  {
    value: '2',
    label: '5 - 10'
  },
  {
    value: '3',
    label: '10+'
  }
];

// ==============================|| CONTACT US - FORM ||============================== //

function ContactForm() {
  const theme = useTheme();
  const [budget, setBudget] = useState(1);
  const handleProjectBudget = (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setBudget(Number(event.target?.value!));
  };

  const [size, setSize] = useState(1);
  const handleCompanySize = (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setSize(Number(event.target?.value!));
  };
  return (
    <Box sx={{ p: { xs: 2.5, sm: 0 } }}>
      <Grid container spacing={5} justifyContent="center">
        <Grid item xs={12} sm={10} lg={9}>
          <Stack alignItems="center" justifyContent="center" spacing={2}>
            <Button variant="text" sx={{ p: 0, textTransform: 'none', '&:hover': { bgcolor: 'transparent' } }}>
              Get In touch
            </Button>
            <Typography align="center" variant="h2">
              Lorem isume dolor elits.
            </Typography>
            <Typography variant="caption" align="center" color="textSecondary" sx={{ maxWidth: '355px' }}>
              The starting point for your next project based on easy-to-customize Material-UI © helps you build apps faster and better.
            </Typography>
          </Stack>
        </Grid>
        <Grid item xs={12} sm={10} lg={9}>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <TextField fullWidth type="text" placeholder="Name" sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }} />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField fullWidth type="text" placeholder="Company Name" sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }} />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField fullWidth type="email" placeholder="Email Address" sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }} />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField fullWidth type="number" placeholder="Phone Number" sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }} />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                select
                fullWidth
                placeholder="Company Size"
                sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }}
                value={size}
                onChange={handleCompanySize}
              >
                {sizes.map((option, index) => (
                  <MenuItem key={index} value={option.value}>
                    {option.label}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                select
                fullWidth
                placeholder="Project Budget"
                sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }}
                value={budget}
                onChange={handleProjectBudget}
              >
                {currencies.map((option, index) => (
                  <MenuItem key={index} value={option.value}>
                    {option.label}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12}>
              <TextField fullWidth multiline rows={4} placeholder="Message" sx={{ '& .MuiOutlinedInput-input': { opacity: 0.5 } }} />
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12} sm={10} lg={9}>
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={{ xs: 1 }}
            justifyContent="space-between"
            alignItems={{ xs: 'stretch', sm: 'center' }}
          >
            <Stack direction="row" alignItems="center" sx={{ ml: -1 }}>
              <Checkbox sx={{ '& .css-1vjb4cj': { borderRadius: '2px' } }} defaultChecked />
              <Typography>
                By submitting this, you agree to the{' '}
                <Typography sx={{ cursor: 'pointer' }} component="span" color={theme.palette.primary.main}>
                  Privacy Policy
                </Typography>{' '}
                and{' '}
                <Typography sx={{ cursor: 'pointer' }} component="span" color={theme.palette.primary.main}>
                  Cookie Policy
                </Typography>{' '}
              </Typography>
            </Stack>
            <Button variant="contained" sx={{ ml: { xs: 0 } }}>
              Submit Now
            </Button>
          </Stack>
        </Grid>
      </Grid>
    </Box>
  );
}

export default ContactForm;
