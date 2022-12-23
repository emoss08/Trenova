import { ReactElement, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

// material-ui
import {
  Button,
  Checkbox,
  Grid,
  InputAdornment,
  List,
  ListItemAvatar,
  ListItemButton,
  ListItemSecondaryAction,
  ListItemText,
  Stack,
  TextField,
  Typography
} from '@mui/material';

// types
import { Address } from 'types/e-commerce';
import { CartCheckoutStateProps } from 'types/cart';

// project imports
import AddressCard from './AddressCard';
import CartDiscount from './CartDiscount';
import OrderSummary from './OrderSummery';
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';

// assets
import { AppstoreOutlined, LeftOutlined, DeleteOutlined } from '@ant-design/icons';

const prodImage = require.context('assets/images/e-commerce', true);

// ==============================|| CHECKOUT BILLING ADDRESS - MAIN ||============================== //

interface BillingAddressProps {
  address: Address[];
  checkout: CartCheckoutStateProps;
  onBack: () => void;
  billingAddressHandler: (billingAddress: Address | null) => void;
  removeProduct: (id: string | number | undefined) => void;
}

const BillingAddress = ({ checkout, onBack, billingAddressHandler, removeProduct, address }: BillingAddressProps) => {
  const [rows, setRows] = useState(checkout.products);

  useEffect(() => {
    setRows(checkout.products);
  }, [checkout.products]);

  let addressResult: ReactElement | ReactElement[] = <></>;
  if (address) {
    addressResult = address.map((data: Address, index: number) => (
      <Grid item xs={12} lg={6} key={index}>
        <AddressCard address={data} billingAddressHandler={billingAddressHandler} />
      </Grid>
    ));
  }

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={8}>
        <Stack spacing={2} alignItems="flex-end">
          <MainCard title="Shipping information">
            <Stack spacing={2}>
              <Grid container spacing={2}>
                {addressResult}
              </Grid>
              <Grid container rowSpacing={2}>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          First Name :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter your first name
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <TextField fullWidth />
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          Last Name :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter your last name
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <TextField fullWidth />
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          Email Id :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter Email id
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <TextField fullWidth type="email" />
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          Date of Birth :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter the date of birth
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <Grid container spacing={2}>
                        <Grid item xs={4}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <TextField
                              fullWidth
                              placeholder="31"
                              InputProps={{
                                endAdornment: (
                                  <InputAdornment position="end" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'flex' } }}>
                                    <AppstoreOutlined />
                                  </InputAdornment>
                                )
                              }}
                            />
                            <Typography>/</Typography>
                          </Stack>
                        </Grid>
                        <Grid item xs={4}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <TextField
                              fullWidth
                              placeholder="12"
                              InputProps={{
                                endAdornment: (
                                  <InputAdornment position="end" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'flex' } }}>
                                    <AppstoreOutlined />
                                  </InputAdornment>
                                )
                              }}
                            />
                            <Typography>/</Typography>
                          </Stack>
                        </Grid>
                        <Grid item xs={4}>
                          <TextField
                            fullWidth
                            placeholder="2021"
                            InputProps={{
                              endAdornment: (
                                <InputAdornment position="end" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'flex' } }}>
                                  <AppstoreOutlined />
                                </InputAdornment>
                              )
                            }}
                          />
                        </Grid>
                      </Grid>
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          Phone number :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter the Phone number
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <Stack direction="row" spacing={2}>
                        <Grid item xs={2}>
                          <TextField placeholder="+91" />
                        </Grid>
                        <Grid item xs={10}>
                          <TextField
                            fullWidth
                            type="number"
                            InputProps={{
                              endAdornment: (
                                <InputAdornment position="end" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'flex' } }}>
                                  <AppstoreOutlined />
                                </InputAdornment>
                              )
                            }}
                          />
                        </Grid>
                      </Stack>
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Grid container alignItems="center" justifyContent="space-between">
                    <Grid item xs={4}>
                      <Stack>
                        <Typography variant="subtitle1" color="textSecondary">
                          City :
                        </Typography>
                        <Typography color="textSecondary" sx={{ opacity: 0.5, display: { xs: 'none', sm: 'block' } }}>
                          Enter City name
                        </Typography>
                      </Stack>
                    </Grid>
                    <Grid item xs={8}>
                      <TextField fullWidth />
                    </Grid>
                  </Grid>
                </Grid>
                <Grid item xs={12}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <Checkbox defaultChecked sx={{ p: 0 }} />
                    <Typography>Save this new address for future shipping</Typography>
                  </Stack>
                </Grid>
                <Grid item xs={12}>
                  <Stack direction="row" spacing={1} alignItems="center" justifyContent="flex-end">
                    <Button variant="outlined" color="secondary">
                      Cancel
                    </Button>
                    <Button variant="contained" color="primary">
                      Save
                    </Button>
                  </Stack>
                </Grid>
              </Grid>
            </Stack>
          </MainCard>
          <Button variant="text" color="secondary" startIcon={<LeftOutlined />} onClick={onBack}>
            <Typography variant="h6" color="textPrimary">
              Back to Cart
            </Typography>
          </Button>
        </Stack>
      </Grid>
      <Grid item xs={12} md={4}>
        <Stack spacing={3}>
          <MainCard>
            <CartDiscount />
          </MainCard>
          <Stack>
            <MainCard title="Order Summery" sx={{ borderRadius: '4px 4px 0 0', borderBottom: 'none' }} content={false}>
              {rows.map((row, index) => (
                <List
                  key={index}
                  component="nav"
                  sx={{
                    '& .MuiListItemButton-root': {
                      '& .MuiListItemSecondaryAction-root': {
                        alignSelf: 'flex-start',
                        ml: 1,
                        position: 'relative',
                        right: 'auto',
                        top: 'auto',
                        transform: 'none'
                      },
                      '& .MuiListItemAvatar-root': { mr: '7px' },
                      py: 0.5,
                      pl: '15px',
                      pr: '8px'
                    },
                    p: 0
                  }}
                >
                  <ListItemButton divider>
                    <ListItemAvatar>
                      <Avatar
                        alt="Avatar"
                        size="lg"
                        variant="rounded"
                        color="secondary"
                        type="combined"
                        src={row.image ? prodImage(`./thumbs/${row.image}`) : ''}
                      />
                    </ListItemAvatar>
                    <ListItemText
                      disableTypography
                      primary={
                        <Typography
                          component={Link}
                          to={`/apps/e-commerce/product-details/${row.id}`}
                          target="_blank"
                          variant="subtitle1"
                          color="textPrimary"
                          sx={{ textDecoration: 'none' }}
                        >
                          {row.name}
                        </Typography>
                      }
                      secondary={
                        <Stack spacing={1}>
                          <Typography color="textSecondary">{row.description}</Typography>
                          <Stack direction="row" alignItems="center" spacing={3}>
                            <Typography>${row.offerPrice}</Typography>
                            <Typography color="textSecondary">{row.quantity} items</Typography>
                          </Stack>
                        </Stack>
                      }
                    />
                    <ListItemSecondaryAction>
                      <IconButton size="medium" color="secondary" sx={{ opacity: 0.5, '&:hover': { bgcolor: 'transparent' } }}>
                        <DeleteOutlined style={{ color: 'grey.500' }} />
                      </IconButton>
                    </ListItemSecondaryAction>
                  </ListItemButton>
                </List>
              ))}
            </MainCard>
            <OrderSummary checkout={checkout} show={false} />
          </Stack>
          <Button variant="contained" fullWidth sx={{ textTransform: 'none' }} onClick={() => billingAddressHandler(null)}>
            Process to Checkout
          </Button>
        </Stack>
      </Grid>
    </Grid>
  );
};

export default BillingAddress;
