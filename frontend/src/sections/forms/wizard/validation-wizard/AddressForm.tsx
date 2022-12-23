// material-ui
import { Button, Checkbox, FormControlLabel, Grid, InputLabel, Stack, Typography, TextField } from '@mui/material';

// third-party
import { useFormik } from 'formik';
import * as yup from 'yup';

// project imports
import AnimateButton from 'components/@extended/AnimateButton';

const validationSchema = yup.object({
  firstName: yup.string().required('First Name is required'),
  lastName: yup.string().required('Last Name is required')
});

// ==============================|| VALIDATION WIZARD - ADDRESS  ||============================== //

export type ShippingData = {
  firstName?: string;
  lastName?: string;
};

interface AddressFormProps {
  shippingData: ShippingData;
  setShippingData: (d: ShippingData) => void;
  handleNext: () => void;
  setErrorIndex: (i: number | null) => void;
}

const AddressForm = ({ shippingData, setShippingData, handleNext, setErrorIndex }: AddressFormProps) => {
  const formik = useFormik({
    initialValues: {
      firstName: shippingData.firstName,
      lastName: shippingData.lastName
    },
    validationSchema,
    onSubmit: (values) => {
      setShippingData({
        firstName: values.firstName,
        lastName: values.lastName
      });
      handleNext();
    }
  });

  return (
    <>
      <Typography variant="h5" gutterBottom sx={{ mb: 2 }}>
        Shipping address
      </Typography>
      <form onSubmit={formik.handleSubmit} id="validation-forms">
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>First Name</InputLabel>
              <TextField
                id="firstName"
                name="firstName"
                placeholder="First Name *"
                value={formik.values.firstName}
                onChange={formik.handleChange}
                error={formik.touched.firstName && Boolean(formik.errors.firstName)}
                helperText={formik.touched.firstName && formik.errors.firstName}
                fullWidth
                autoComplete="given-name"
              />
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>Last Name</InputLabel>
              <TextField
                id="lastName"
                name="lastName"
                placeholder="Last Name *"
                value={formik.values.lastName}
                onChange={formik.handleChange}
                error={formik.touched.lastName && Boolean(formik.errors.lastName)}
                helperText={formik.touched.lastName && formik.errors.lastName}
                fullWidth
                autoComplete="family-name"
              />
            </Stack>
          </Grid>
          <Grid item xs={12}>
            <Stack spacing={0.5}>
              <InputLabel>Address 1</InputLabel>
              <TextField id="address1" name="address1" placeholder="Address line 1" fullWidth autoComplete="shipping address-line1" />
            </Stack>
          </Grid>
          <Grid item xs={12}>
            <Stack spacing={0.5}>
              <InputLabel>Address 2</InputLabel>
              <TextField id="address2" name="address2" placeholder="Address line 2" fullWidth autoComplete="shipping address-line2" />
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>Enter City</InputLabel>
              <TextField id="city" name="city" placeholder="City" fullWidth autoComplete="shipping address-level2" />
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>Enter State</InputLabel>
              <TextField id="state" name="state" placeholder="State/Province/Region" fullWidth />
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>Zip Code</InputLabel>
              <TextField id="zip" name="zip" placeholder="Zip / Postal code" fullWidth autoComplete="shipping postal-code" />
            </Stack>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Stack spacing={0.5}>
              <InputLabel>Enter Country</InputLabel>
              <TextField id="country" name="country" placeholder="Country" fullWidth autoComplete="shipping country" />
            </Stack>
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox color="primary" name="saveAddress" value="yes" />}
              label="Use this address for payment details"
            />
          </Grid>
          <Grid item xs={12}>
            <Stack direction="row" justifyContent="flex-end">
              <AnimateButton>
                <Button variant="contained" sx={{ my: 3, ml: 1 }} type="submit" onClick={() => setErrorIndex(0)}>
                  Next
                </Button>
              </AnimateButton>
            </Stack>
          </Grid>
        </Grid>
      </form>
    </>
  );
};

export default AddressForm;
