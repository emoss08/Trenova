// third-party
import { Chance } from 'chance';
import { filter } from 'lodash';

// project imports
import services from 'utils/mockAdapter';

// types
import { CartProductStateProps, ProductCardProps } from 'types/cart';

const chance = new Chance();

let subtotal: number;
let result;

let latestProducts: CartProductStateProps[];
let newProduct: CartProductStateProps;
let inCartProduct: CartProductStateProps[];
let oldSubTotal;
let amount;
let newShipping;

// ==============================|| MOCK SERVICES ||============================== //

services.onPost('/api/cart/add').reply((config) => {
  try {
    const { product, products } = JSON.parse(config.data);

    newProduct = { ...product!, itemId: chance.timestamp() };
    subtotal = newProduct?.quantity * newProduct.offerPrice;

    inCartProduct = filter(products, {
      id: newProduct.id,
      color: newProduct.color,
      size: newProduct.size
    });
    if (inCartProduct && inCartProduct.length > 0) {
      const newProducts = products.map((item: CartProductStateProps) => {
        if (newProduct?.id === item.id && newProduct?.color === item.color) {
          return { ...newProduct, quantity: newProduct.quantity + inCartProduct[0].quantity };
        }
        return item;
      });
      latestProducts = newProducts;
    } else {
      latestProducts = [...products, newProduct];
    }

    return [200, { products: latestProducts, subtotal }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/remove').reply((config) => {
  try {
    const { id, products } = JSON.parse(config.data);

    result = filter(products, { itemId: id });
    subtotal = result[0].quantity * result[0].offerPrice;

    const newProducts = filter(products, (item) => item.itemId !== id);

    return [200, { products: newProducts, subtotal }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/update').reply((config) => {
  try {
    const { id, quantity, products } = JSON.parse(config.data);

    result = filter(products, { itemId: id });
    subtotal = quantity! * result[0].offerPrice;
    oldSubTotal = 0;

    latestProducts = products.map((item: ProductCardProps) => {
      if (id === item.itemId) {
        oldSubTotal = item.quantity * (item.offerPrice || 0);
        return { ...item, quantity: quantity! };
      }
      return item;
    });

    return [200, { products: latestProducts, oldSubTotal, subtotal }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/billing-address').reply((config) => {
  try {
    const { address } = JSON.parse(config.data);
    return [200, { billing: address! }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/discount').reply((config) => {
  try {
    const { total, code } = JSON.parse(config.data);
    amount = 0;
    if (total > 0) {
      switch (code) {
        case 'MANTIS50':
          amount = chance.integer({ min: 1, max: total < 49 ? total : 49 });
          break;
        case 'FLAT05':
          amount = total < 5 ? total : 5;
          break;
        case 'SUB150':
          amount = total < 150 ? total : 150;
          break;
        case 'UPTO200':
          amount = chance.integer({ min: 1, max: total < 199 ? total : 199 });
          break;
        default:
          amount = 0;
      }
    }

    return [200, { amount }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/shipping-charge').reply((config) => {
  try {
    const { shipping, charge } = JSON.parse(config.data);
    newShipping = 0;
    if (shipping > 0 && charge === 'free') {
      newShipping = -5;
    }
    if (charge === 'fast') {
      newShipping = 5;
    }

    return [200, { shipping: charge === 'fast' ? 5 : 0, newShipping, type: charge! }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/payment-method').reply((config) => {
  try {
    const { method } = JSON.parse(config.data);
    return [200, { method: method! }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/payment-card').reply((config) => {
  try {
    const { card } = JSON.parse(config.data);
    return [200, { card: card! }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/cart/reset').reply(200, {});
