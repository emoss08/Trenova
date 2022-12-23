import { useEffect, useState, ReactNode } from 'react';

// material-ui
import { styled, Theme, useTheme } from '@mui/material/styles';
import { Grid, Stack, Tab, Tabs, Typography } from '@mui/material';

// types
import { CartStateProps, DefaultRootStateProps } from 'types/cart';
import { Address, TabsProps } from 'types/e-commerce';

// project imports
import Avatar from 'components/@extended/Avatar';
import BillingAddress from 'sections/apps/e-commerce/checkout/BillingAddress';
import Cart from 'sections/apps/e-commerce/checkout/Cart';
import CartEmpty from 'sections/apps/e-commerce/checkout/CartEmpty';
import Payment from 'sections/apps/e-commerce/checkout/Payment';
import MainCard from 'components/MainCard';
import { openSnackbar } from 'store/reducers/snackbar';
import { useDispatch, useSelector } from 'store';
import { getAddresses, editAddress } from 'store/reducers/product';
import { removeProduct, setBackStep, setBillingAddress, setNextStep, setShippingCharge, setStep, updateProduct } from 'store/reducers/cart';

// assets
import { CheckOutlined } from '@ant-design/icons';

interface StyledProps {
  theme: Theme;
  value: number;
  cart: CartStateProps;
  disabled?: boolean;
  icon?: ReactNode;
  label?: ReactNode;
}

interface TabOptionProps {
  label: string;
}

const StyledTab = styled((props) => <Tab {...props} />)(({ theme, value, cart, ...others }: StyledProps) => ({
  minHeight: 'auto',
  minWidth: 250,
  padding: 16,
  display: 'flex',
  flexDirection: 'row',
  alignItems: 'flex-start',
  textAlign: 'left',
  justifyContent: 'flex-start',
  '&:after': {
    backgroundColor: 'transparent !important'
  },

  '& > svg': {
    marginBottom: '0px !important',
    marginRight: 10,
    marginTop: 2,
    height: 20,
    width: 20
  },
  [theme.breakpoints.down('md')]: {
    minWidth: 'auto'
  }
}));

// tabs option
const tabsOption: TabOptionProps[] = [
  {
    label: 'Cart'
  },
  {
    label: 'Shipping Information'
  },
  {
    label: 'Payment'
  }
];

// tabs
function TabPanel({ children, value, index, ...other }: TabsProps) {
  return (
    <div role="tabpanel" hidden={value !== index} id={`simple-tabpanel-${index}`} aria-labelledby={`simple-tab-${index}`} {...other}>
      {value === index && <div>{children}</div>}
    </div>
  );
}

// ==============================|| PRODUCT - CHECKOUT MAIN ||============================== //

const Checkout = () => {
  const theme = useTheme();
  const cart = useSelector((state: DefaultRootStateProps) => state.cart);
  const dispatch = useDispatch();

  const isCart = cart.checkout.products && cart.checkout.products.length > 0;

  const [value, setValue] = useState(cart.checkout.step > 2 ? 2 : cart.checkout.step);
  const [billing, setBilling] = useState(cart.checkout.billing);
  const [address, setAddress] = useState<Address[]>([]);
  const { addresses } = useSelector((state) => state.product);

  useEffect(() => {
    setAddress(addresses);
  }, [addresses]);

  useEffect(() => {
    dispatch(getAddresses());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const editBillingAddress = (addressEdit: Address) => {
    dispatch(editAddress(addressEdit));
  };

  const handleChange = (newValue: number) => {
    setValue(newValue);
    dispatch(setStep(newValue));
  };

  useEffect(() => {
    setValue(cart.checkout.step > 2 ? 2 : cart.checkout.step);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cart.checkout.step]);

  const removeCartProduct = (id: string | number | undefined) => {
    dispatch(removeProduct(id, cart.checkout.products));
    dispatch(
      openSnackbar({
        open: true,
        message: 'Update Cart Success',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    );
  };

  const updateQuantity = (id: string | number | undefined, quantity: number) => {
    dispatch(updateProduct(id, quantity, cart.checkout.products));
  };

  const onNext = () => {
    dispatch(setNextStep());
  };

  const onBack = () => {
    dispatch(setBackStep());
  };

  const billingAddressHandler = (addressBilling: Address | null) => {
    if (billing !== null || addressBilling !== null) {
      if (addressBilling !== null) {
        setBilling(addressBilling);
      }

      dispatch(setBillingAddress(addressBilling !== null ? addressBilling : billing));
      onNext();
    } else {
      dispatch(
        openSnackbar({
          open: true,
          message: 'Please select delivery address',
          variant: 'alert',
          alert: {
            color: 'error'
          },
          close: false
        })
      );
    }
  };

  const handleShippingCharge = (type: string) => {
    dispatch(setShippingCharge(type, cart.checkout.shipping));
  };

  return (
    <Stack spacing={2}>
      <MainCard content={false}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Tabs
              value={value}
              onChange={(e, newValue) => handleChange(newValue)}
              aria-label="icon label tabs example"
              variant="scrollable"
              sx={{
                '& .MuiTabs-flexContainer': {
                  borderBottom: 'none'
                },
                '& .MuiTabs-indicator': {
                  display: 'none'
                },
                '& .MuiButtonBase-root + .MuiButtonBase-root': {
                  position: 'relative',
                  overflow: 'visible',
                  ml: 2,
                  '&:after': {
                    content: '""',
                    bgcolor: '#ccc',
                    width: 1,
                    height: 'calc(100% - 16px)',
                    position: 'absolute',
                    top: 8,
                    left: -8
                  }
                }
              }}
            >
              {tabsOption.map((tab, index) => (
                <StyledTab
                  theme={theme}
                  value={index}
                  cart={cart}
                  disabled={index > cart.checkout.step}
                  key={index}
                  label={
                    <Grid container>
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <Avatar
                          type={index !== cart.checkout.step ? 'combined' : 'filled'}
                          size="xs"
                          color={index > cart.checkout.step ? 'secondary' : 'primary'}
                        >
                          {index === cart.checkout.step ? index + 1 : <CheckOutlined />}
                        </Avatar>
                        <Typography color={index > cart.checkout.step ? 'textSecondary' : 'inherit'}>{tab.label}</Typography>
                      </Stack>
                    </Grid>
                  }
                />
              ))}
            </Tabs>
          </Grid>
        </Grid>
      </MainCard>
      <Grid container>
        <Grid item xs={12}>
          <TabPanel value={value} index={0}>
            {isCart && <Cart checkout={cart.checkout} onNext={onNext} removeProduct={removeCartProduct} updateQuantity={updateQuantity} />}
            {!isCart && <CartEmpty />}
          </TabPanel>
          <TabPanel value={value} index={1}>
            <BillingAddress
              checkout={cart.checkout}
              onBack={onBack}
              removeProduct={removeCartProduct}
              billingAddressHandler={billingAddressHandler}
              address={address}
            />
          </TabPanel>
          <TabPanel value={value} index={2}>
            <Payment
              checkout={cart.checkout}
              onBack={onBack}
              onNext={onNext}
              handleShippingCharge={handleShippingCharge}
              removeProduct={removeCartProduct}
              editAddress={editBillingAddress}
            />
          </TabPanel>
        </Grid>
      </Grid>
    </Stack>
  );
};

export default Checkout;
