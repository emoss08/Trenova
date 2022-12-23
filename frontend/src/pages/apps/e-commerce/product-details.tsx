import { useEffect, useState, SyntheticEvent } from 'react';
import { Link, useParams } from 'react-router-dom';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Chip, Divider, Grid, Stack, Tab, Tabs, Typography } from '@mui/material';

// types
import { TabsProps } from 'types/e-commerce';

// project imports
import FloatingCart from 'components/cards/e-commerce/FloatingCart';
import ProductFeatures from 'sections/apps/e-commerce/product-details/ProductFeatures';
import ProductImages from 'sections/apps/e-commerce/product-details/ProductImages';
import ProductInfo from 'sections/apps/e-commerce/product-details/ProductInfo';
import ProductReview from 'sections/apps/e-commerce/product-details/ProductReview';
import ProductSpecifications from 'sections/apps/e-commerce/product-details/ProductSpecifications';
import RelatedProducts from 'sections/apps/e-commerce/product-details/RelatedProducts';
import MainCard from 'components/MainCard';
import { useDispatch, useSelector } from 'store';
import { getProduct } from 'store/reducers/product';
import { resetCart } from 'store/reducers/cart';

function TabPanel({ children, value, index, ...other }: TabsProps) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`product-details-tabpanel-${index}`}
      aria-labelledby={`product-details-tab-${index}`}
      {...other}
    >
      {value === index && <Box>{children}</Box>}
    </div>
  );
}

function a11yProps(index: number) {
  return {
    id: `product-details-tab-${index}`,
    'aria-controls': `product-details-tabpanel-${index}`
  };
}

// ==============================|| PRODUCT DETAILS - MAIN ||============================== //

const ProductDetails = () => {
  const theme = useTheme();
  const { id } = useParams();

  const dispatch = useDispatch();
  const cart = useSelector((state) => state.cart);

  // product description tabs
  const [value, setValue] = useState(0);

  const handleChange = (event: SyntheticEvent, newValue: number) => {
    setValue(newValue);
  };
  const { product } = useSelector((state) => state.product);

  useEffect(() => {
    dispatch(getProduct(id));

    // clear cart if complete order
    if (cart.checkout.step > 2) {
      dispatch(resetCart());
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  return (
    <>
      {product && product.id === Number(id) && (
        <Grid container spacing={2}>
          <Grid item xs={12}>
            <MainCard>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <ProductImages />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <ProductInfo product={product} />
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} md={7} xl={8}>
            <MainCard>
              <Stack spacing={3}>
                <Stack>
                  <Tabs
                    value={value}
                    indicatorColor="primary"
                    onChange={handleChange}
                    aria-label="product description tabs example"
                    variant="scrollable"
                  >
                    <Tab component={Link} to="#" label="Features" {...a11yProps(0)} />
                    <Tab component={Link} to="#" label="Specifications" {...a11yProps(1)} />
                    <Tab component={Link} to="#" label="Overview" {...a11yProps(2)} />
                    <Tab
                      component={Link}
                      to="#"
                      label={
                        <Stack direction="row" alignItems="center">
                          Reviews{' '}
                          <Chip
                            label={String(product.offerPrice?.toFixed(0))}
                            size="small"
                            sx={{
                              ml: 0.5,
                              color: value === 3 ? theme.palette.primary.main : theme.palette.grey[0],
                              bgcolor: value === 3 ? theme.palette.primary.lighter : theme.palette.grey[400],
                              borderRadius: '10px'
                            }}
                          />
                        </Stack>
                      }
                      {...a11yProps(3)}
                    />
                  </Tabs>
                  <Divider />
                </Stack>
                <TabPanel value={value} index={0}>
                  <ProductFeatures />
                </TabPanel>
                <TabPanel value={value} index={1}>
                  <ProductSpecifications />
                </TabPanel>
                <TabPanel value={value} index={2}>
                  <Stack spacing={2.5}>
                    <Typography color="textSecondary">
                      Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industrys standard
                      dummy text ever since the 1500s,{' '}
                      <Typography component="span" variant="subtitle1">
                        {' '}
                        &ldquo;When an unknown printer took a galley of type and scrambled it to make a type specimen book.&rdquo;
                      </Typography>{' '}
                      It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially
                      unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and
                      more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.
                    </Typography>
                    <Typography color="textSecondary">
                      It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently
                      with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.
                    </Typography>
                  </Stack>
                </TabPanel>
                <TabPanel value={value} index={3}>
                  <ProductReview product={product} />
                </TabPanel>
              </Stack>
            </MainCard>
          </Grid>
          <Grid item xs={12} md={5} xl={4} sx={{ position: 'relative' }}>
            <MainCard
              title="Related Products"
              sx={{
                height: 'calc(100% - 16px)',
                position: { xs: 'relative', md: 'absolute' },
                top: '16px',
                left: { xs: 0, md: 16 },
                right: 0
              }}
              content={false}
            >
              <RelatedProducts id={id} />
            </MainCard>
          </Grid>
        </Grid>
      )}
      <FloatingCart />
    </>
  );
};

export default ProductDetails;
