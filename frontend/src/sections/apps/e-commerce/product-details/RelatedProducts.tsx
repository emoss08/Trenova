import { ReactElement, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';

// material-ui
import { useTheme } from '@mui/material/styles';
import {
  Button,
  Grid,
  List,
  ListItemAvatar,
  ListItemButton,
  ListItemSecondaryAction,
  ListItemText,
  Rating,
  Stack,
  Typography
} from '@mui/material';

// types
import { Products } from 'types/e-commerce';

// project imports
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';
import { useDispatch, useSelector } from 'store';
import { getRelatedProducts } from 'store/reducers/product';
import { openSnackbar } from 'store/reducers/snackbar';
import SimpleBar from 'components/third-party/SimpleBar';

// assets
import { HeartFilled, HeartOutlined, StarFilled, StarOutlined } from '@ant-design/icons';

const prodImage = require.context('assets/images/e-commerce', true);

const ListProduct = ({ product }: { product: Products }) => {
  const theme = useTheme();
  const history = useNavigate();
  const dispatch = useDispatch();

  const [wishlisted, setWishlisted] = useState<boolean>(false);
  const addToFavourite = () => {
    setWishlisted(!wishlisted);
    dispatch(
      openSnackbar({
        open: true,
        message: 'Added to favourites',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    );
  };

  const linkHandler = (id?: string | number) => {
    history(`/apps/e-commerce/product-details/${id}`);
  };

  return (
    <ListItemButton divider onClick={() => linkHandler(product.id)}>
      <ListItemAvatar>
        <Avatar
          alt="Avatar"
          size="xl"
          color="secondary"
          variant="rounded"
          type="combined"
          src={product.image ? prodImage(`./thumbs/${product.image}`) : ''}
          sx={{ borderColor: theme.palette.divider, mr: 1 }}
        />
      </ListItemAvatar>
      <ListItemText
        disableTypography
        primary={<Typography variant="h5">{product.name}</Typography>}
        secondary={
          <Stack spacing={1}>
            <Typography color="textSecondary">{product.description}</Typography>
            <Stack spacing={1}>
              <Stack direction="row" alignItems="center" spacing={0.5}>
                <Typography variant="h5">{product.salePrice ? `$${product.salePrice}` : `$${product.offerPrice}`}</Typography>
                {product.salePrice && (
                  <Typography variant="h6" color="textSecondary" sx={{ textDecoration: 'line-through' }}>
                    ${product.offerPrice}
                  </Typography>
                )}
              </Stack>
              <Rating
                name="simple-controlled"
                value={product.rating! < 4 ? product.rating! + 1 : product.rating}
                icon={<StarFilled style={{ fontSize: 'small' }} />}
                emptyIcon={<StarOutlined style={{ fontSize: 'small' }} />}
                readOnly
                precision={0.1}
              />
            </Stack>
          </Stack>
        }
      />
      <ListItemSecondaryAction>
        <IconButton
          size="medium"
          color="secondary"
          sx={{ opacity: wishlisted ? 1 : 0.5, '&:hover': { bgcolor: 'transparent' } }}
          onClick={addToFavourite}
        >
          {wishlisted ? (
            <HeartFilled style={{ fontSize: '1.15rem', color: theme.palette.error.main }} />
          ) : (
            <HeartOutlined style={{ fontSize: '1.15rem' }} />
          )}
        </IconButton>
      </ListItemSecondaryAction>
    </ListItemButton>
  );
};

// ==============================|| PRODUCT DETAILS - RELATED PRODUCTS ||============================== //

const RelatedProducts = ({ id }: { id?: string }) => {
  const dispatch = useDispatch();
  const [related, setRelated] = useState<Products[]>([]);
  const { relatedProducts } = useSelector((state) => state.product);

  useEffect(() => {
    setRelated(relatedProducts);
  }, [relatedProducts]);

  useEffect(() => {
    dispatch(getRelatedProducts(id));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  let productResult: ReactElement | ReactElement[] = <></>;
  if (related) {
    productResult = (
      <List
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
        {related.map((product: Products, index) => (
          <ListProduct key={index} product={product} />
        ))}
      </List>
    );
  }

  return (
    <SimpleBar sx={{ height: { xs: '100%', md: 'calc(100% - 62px)' } }}>
      <Grid item>
        <Stack>
          {productResult}
          <Button color="secondary" variant="outlined" sx={{ mx: 2, my: 4, textTransform: 'none' }}>
            View all Products
          </Button>
        </Stack>
      </Grid>
    </SimpleBar>
  );
};

export default RelatedProducts;
