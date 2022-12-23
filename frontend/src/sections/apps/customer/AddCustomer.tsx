import { useEffect, useState, ChangeEvent } from 'react';
import { useDispatch } from 'react-redux';

// material-ui
import { useTheme } from '@mui/material/styles';
import {
  Box,
  Button,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormControlLabel,
  FormLabel,
  Grid,
  FormHelperText,
  InputLabel,
  ListItemText,
  MenuItem,
  OutlinedInput,
  Select,
  SelectChangeEvent,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography
} from '@mui/material';
import { LocalizationProvider } from '@mui/x-date-pickers';

import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';

// third-party
import _ from 'lodash';
import * as Yup from 'yup';
import { useFormik, Form, FormikProvider, FormikValues } from 'formik';

// project imports
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';
import { openSnackbar } from 'store/reducers/snackbar';

// assets
import { CameraOutlined, DeleteFilled } from '@ant-design/icons';

const avatarImage = require.context('assets/images/users', true);

// constant
const getInitialValues = (customer: FormikValues | null) => {
  const newCustomer = {
    name: '',
    email: '',
    location: '',
    orderStatus: ''
  };

  if (customer) {
    newCustomer.name = customer.fatherName;
    newCustomer.location = customer.address;
    return _.merge({}, newCustomer, customer);
  }

  return newCustomer;
};

const allStatus = ['Complicated', 'Single', 'Relationship'];

// ==============================|| CUSTOMER ADD / EDIT / DELETE ||============================== //

export interface Props {
  customer?: any;
  onCancel: () => void;
}

const AddCustomer = ({ customer, onCancel }: Props) => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const isCreating = !customer;

  const [selectedImage, setSelectedImage] = useState<File | undefined>(undefined);
  const [avatar, setAvatar] = useState<string | undefined>(
    avatarImage(`./avatar-${isCreating && !customer?.avatar ? 1 : customer.avatar}.png`)
  );

  useEffect(() => {
    if (selectedImage) {
      setAvatar(URL.createObjectURL(selectedImage));
    }
  }, [selectedImage]);

  const CustomerSchema = Yup.object().shape({
    name: Yup.string().max(255).required('Name is required'),
    orderStatus: Yup.string().required('Name is required'),
    email: Yup.string().max(255).required('Email is required').email('Must be a valid email'),
    location: Yup.string().max(500)
  });

  const deleteHandler = () => {
    // dispatch(deleteCustomer(customer?.id)); - delete
    dispatch(
      openSnackbar({
        open: true,
        message: 'Customer deleted successfully.',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    );
    onCancel();
  };

  const formik = useFormik({
    initialValues: getInitialValues(customer!),
    validationSchema: CustomerSchema,
    onSubmit: (values, { setSubmitting }) => {
      try {
        // const newCustomer = {
        //   name: values.name,
        //   email: values.email,
        //   location: values.location,
        //   orderStatus: values.orderStatus
        // };

        if (customer) {
          // dispatch(updateCustomer(customer.id, newCustomer)); - update
          dispatch(
            openSnackbar({
              open: true,
              message: 'Customer update successfully.',
              variant: 'alert',
              alert: {
                color: 'success'
              },
              close: false
            })
          );
        } else {
          // dispatch(createCustomer(newCustomer)); - add
          dispatch(
            openSnackbar({
              open: true,
              message: 'Customer added successfully.',
              variant: 'alert',
              alert: {
                color: 'success'
              },
              close: false
            })
          );
        }

        setSubmitting(false);
        onCancel();
      } catch (error) {
        console.error(error);
      }
    }
  });

  const { errors, touched, handleSubmit, isSubmitting, getFieldProps, setFieldValue } = formik;

  return (
    <FormikProvider value={formik}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Form autoComplete="off" noValidate onSubmit={handleSubmit}>
          <DialogTitle>{customer ? 'Edit Customer' : 'New Customer'}</DialogTitle>
          <Divider />
          <DialogContent sx={{ p: 2.5 }}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={3}>
                <Stack direction="row" justifyContent="center" sx={{ mt: 3 }}>
                  <FormLabel
                    htmlFor="change-avtar"
                    sx={{
                      position: 'relative',
                      borderRadius: '50%',
                      overflow: 'hidden',
                      '&:hover .MuiBox-root': { opacity: 1 },
                      cursor: 'pointer'
                    }}
                  >
                    <Avatar alt="Avatar 1" src={avatar} sx={{ width: 72, height: 72, border: '1px dashed' }} />
                    <Box
                      sx={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        backgroundColor: theme.palette.mode === 'dark' ? 'rgba(255, 255, 255, .75)' : 'rgba(0,0,0,.65)',
                        width: '100%',
                        height: '100%',
                        opacity: 0,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                      }}
                    >
                      <Stack spacing={0.5} alignItems="center">
                        <CameraOutlined style={{ color: theme.palette.secondary.lighter, fontSize: '2rem' }} />
                        <Typography sx={{ color: 'secondary.lighter' }}>Upload</Typography>
                      </Stack>
                    </Box>
                  </FormLabel>
                  <TextField
                    type="file"
                    id="change-avtar"
                    label="Outlined"
                    variant="outlined"
                    sx={{ display: 'none' }}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setSelectedImage(e.target.files?.[0])}
                  />
                </Stack>
              </Grid>
              <Grid item xs={12} md={8}>
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <Stack spacing={1.25}>
                      <InputLabel htmlFor="customer-name">Name</InputLabel>
                      <TextField
                        fullWidth
                        id="customer-name"
                        placeholder="Enter Customer Name"
                        {...getFieldProps('name')}
                        error={Boolean(touched.name && errors.name)}
                        helperText={touched.name && errors.name}
                      />
                    </Stack>
                  </Grid>
                  <Grid item xs={12}>
                    <Stack spacing={1.25}>
                      <InputLabel htmlFor="customer-email">Email</InputLabel>
                      <TextField
                        fullWidth
                        id="customer-email"
                        placeholder="Enter Customer Email"
                        {...getFieldProps('email')}
                        error={Boolean(touched.email && errors.email)}
                        helperText={touched.email && errors.email}
                      />
                    </Stack>
                  </Grid>
                  <Grid item xs={12}>
                    <Stack spacing={1.25}>
                      <InputLabel htmlFor="customer-orderStatus">Status</InputLabel>
                      <FormControl fullWidth>
                        <Select
                          id="column-hiding"
                          displayEmpty
                          {...getFieldProps('orderStatus')}
                          onChange={(event: SelectChangeEvent<string>) => setFieldValue('orderStatus', event.target.value as string)}
                          input={<OutlinedInput id="select-column-hiding" placeholder="Sort by" />}
                          renderValue={(selected) => {
                            if (!selected) {
                              return <Typography variant="subtitle1">Select Status</Typography>;
                            }

                            return <Typography variant="subtitle2">{selected}</Typography>;
                          }}
                        >
                          {allStatus.map((column: any) => (
                            <MenuItem key={column} value={column}>
                              <ListItemText primary={column} />
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                      {touched.orderStatus && errors.orderStatus && (
                        <FormHelperText error id="standard-weight-helper-text-email-login" sx={{ pl: 1.75 }}>
                          {errors.orderStatus}
                        </FormHelperText>
                      )}
                    </Stack>
                  </Grid>
                  <Grid item xs={12}>
                    <Stack spacing={1.25}>
                      <InputLabel htmlFor="customer-location">Location</InputLabel>
                      <TextField
                        fullWidth
                        id="customer-location"
                        multiline
                        rows={2}
                        placeholder="Enter Location"
                        {...getFieldProps('location')}
                        error={Boolean(touched.location && errors.location)}
                        helperText={touched.location && errors.location}
                      />
                    </Stack>
                  </Grid>
                  <Grid item xs={12}>
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                      <Stack spacing={0.5}>
                        <Typography variant="subtitle1">Make Contact Info Public</Typography>
                        <Typography variant="caption" color="textSecondary">
                          Means that anyone viewing your profile will be able to see your contacts details
                        </Typography>
                      </Stack>
                      <FormControlLabel control={<Switch defaultChecked sx={{ mt: 0 }} />} label="" labelPlacement="start" />
                    </Stack>
                    <Divider sx={{ my: 2 }} />
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                      <Stack spacing={0.5}>
                        <Typography variant="subtitle1">Available to hire</Typography>
                        <Typography variant="caption" color="textSecondary">
                          Toggling this will let your teammates know that you are available for acquiring new projects
                        </Typography>
                      </Stack>
                      <FormControlLabel control={<Switch sx={{ mt: 0 }} />} label="" labelPlacement="start" />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </DialogContent>
          <Divider />
          <DialogActions sx={{ p: 2.5 }}>
            <Grid container justifyContent="space-between" alignItems="center">
              <Grid item>
                {!isCreating && (
                  <Tooltip title="Delete Customer" placement="top">
                    <IconButton onClick={deleteHandler} size="large" color="error">
                      <DeleteFilled />
                    </IconButton>
                  </Tooltip>
                )}
              </Grid>
              <Grid item>
                <Stack direction="row" spacing={2} alignItems="center">
                  <Button color="error" onClick={onCancel}>
                    Cancel
                  </Button>
                  <Button type="submit" variant="contained" disabled={isSubmitting}>
                    {customer ? 'Edit' : 'Add'}
                  </Button>
                </Stack>
              </Grid>
            </Grid>
          </DialogActions>
        </Form>
      </LocalizationProvider>
    </FormikProvider>
  );
};

export default AddCustomer;
