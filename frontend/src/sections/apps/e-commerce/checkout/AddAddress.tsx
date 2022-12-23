import { forwardRef } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import {
  Button,
  Dialog,
  FormControl,
  FormControlLabel,
  FormHelperText,
  Grid,
  Radio,
  RadioGroup,
  Stack,
  Switch,
  TextField,
  Zoom,
  ZoomProps
} from '@mui/material';

// third-party
import { useFormik } from 'formik';
import * as yup from 'yup';

// types
import { Address } from 'types/e-commerce';

// project imports
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import AnimateButton from 'components/@extended/AnimateButton';
import { useDispatch } from 'store';
import { openSnackbar } from 'store/reducers/snackbar';

// assets
import { CloseCircleTwoTone } from '@ant-design/icons';

const validationSchema = yup.object({
  destination: yup.string().required('Color selection is required'),
  name: yup.string().required('Name is required'),
  building: yup.string().required('Building no/name is required'),
  street: yup.string().required('Street Address is required'),
  city: yup.string().required('City is required'),
  state: yup.string().required('State is required'),
  post: yup.string().required('Area code is required'),
  phone: yup.string().required('Contact no is required')
});

const Transition = forwardRef((props: ZoomProps, ref) => <Zoom ref={ref} {...props} />);

// ==============================|| CHECKOUT BILLING ADDRESS - ADD NEW ADDRESS ||============================== //

interface AddAddressProps {
  address: Address;
  open: boolean;
  handleClose: () => void;
  editAddress: (address: Address) => void;
}

const AddAddress = ({ address, open, handleClose, editAddress }: AddAddressProps) => {
  const theme = useTheme();
  const dispatch = useDispatch();

  const edit = address && address.id;

  const formik = useFormik({
    enableReinitialize: true,
    initialValues: {
      destination: edit ? address.destination : '',
      name: edit ? address.name : '',
      building: edit ? address.building : '',
      street: edit ? address.street : '',
      city: edit ? address.city : '',
      state: edit ? address.state : '',
      country: edit ? address.country : '',
      post: edit ? address.post : '',
      phone: edit ? address.phone : '',
      isDefault: edit ? address.isDefault : false
    },
    validationSchema,
    onSubmit: (values) => {
      editAddress({ ...values, id: address.id });
      handleClose();
      dispatch(
        openSnackbar({
          open: true,
          message: 'Submit Success',
          variant: 'alert',
          alert: {
            color: 'success'
          },
          close: false
        })
      );
    }
  });

  return (
    <Dialog
      open={open}
      TransitionComponent={Transition}
      keepMounted
      onClose={handleClose}
      aria-labelledby="alert-dialog-slide-title"
      aria-describedby="alert-dialog-slide-description"
      sx={{
        '& .MuiDialog-paper': {
          p: 0
        }
      }}
    >
      {open && (
        <MainCard
          title="Edit Billing Address"
          secondary={
            <IconButton onClick={handleClose} size="large">
              <CloseCircleTwoTone style={{ fontSize: 'small' }} />
            </IconButton>
          }
        >
          <form onSubmit={formik.handleSubmit}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  id="name"
                  name="name"
                  label="Name"
                  value={formik.values.name}
                  onChange={formik.handleChange}
                  error={formik.touched.name && Boolean(formik.errors.name)}
                  helperText={formik.touched.name && formik.errors.name}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <FormControl>
                  <RadioGroup
                    row
                    aria-label="destination"
                    value={formik.values.destination}
                    onChange={formik.handleChange}
                    name="destination"
                    id="destination"
                  >
                    <FormControlLabel
                      value="home"
                      control={
                        <Radio
                          sx={{
                            color: theme.palette.primary.main,
                            '&.Mui-checked': { color: theme.palette.primary.main }
                          }}
                        />
                      }
                      label="Home"
                    />
                    <FormControlLabel
                      value="office"
                      control={
                        <Radio
                          sx={{
                            color: theme.palette.secondary.main,
                            '&.Mui-checked': { color: theme.palette.secondary.main }
                          }}
                        />
                      }
                      label="Office"
                    />
                  </RadioGroup>
                </FormControl>
                {formik.errors.destination && (
                  <FormHelperText error id="standard-weight-helper-text-name-login">
                    {' '}
                    {formik.errors.destination}{' '}
                  </FormHelperText>
                )}
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  id="building"
                  name="building"
                  label="Building No./Name"
                  value={formik.values.building}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.building)}
                  helperText={formik.touched.building && formik.errors.building}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  id="street"
                  name="street"
                  label="Street"
                  value={formik.values.street}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.street)}
                  helperText={formik.touched.street && formik.errors.street}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  id="city"
                  name="city"
                  label="City"
                  value={formik.values.city}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.city)}
                  helperText={formik.touched.city && formik.errors.city}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  id="state"
                  name="state"
                  label="State"
                  value={formik.values.state}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.state)}
                  helperText={formik.touched.state && formik.errors.state}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  id="country"
                  name="country"
                  label="Country"
                  value={formik.values.country}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.country)}
                  helperText={formik.touched.country && formik.errors.country}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  id="post"
                  name="post"
                  label="Area Code"
                  value={formik.values.post}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.post)}
                  helperText={formik.touched.post && formik.errors.post}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  id="phone"
                  name="phone"
                  label="Contact"
                  value={formik.values.phone}
                  onChange={formik.handleChange}
                  error={formik.touched.building && Boolean(formik.errors.phone)}
                  helperText={formik.touched.phone && formik.errors.phone}
                />
              </Grid>
              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      id="isDefault"
                      name="isDefault"
                      checked={formik.values.isDefault}
                      onChange={formik.handleChange}
                      inputProps={{ 'aria-label': 'controlled' }}
                    />
                  }
                  label="Default"
                />
              </Grid>
              <Grid item xs={12}>
                <Stack direction="row" spacing={1} justifyContent="flex-end">
                  <Button color="error" onClick={handleClose}>
                    Cancel
                  </Button>
                  <AnimateButton>
                    <Button variant="contained" type="submit">
                      Submit
                    </Button>
                  </AnimateButton>
                </Stack>
              </Grid>
            </Grid>
          </form>
        </MainCard>
      )}
    </Dialog>
  );
};

export default AddAddress;
