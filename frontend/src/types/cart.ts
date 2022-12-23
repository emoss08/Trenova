// import { KeyedObject } from 'types';
export type KeyedObject = {
  [key: string]: string | number | KeyedObject | any;
};

export interface CartStateProps {
  checkout: CartCheckoutStateProps;
  error: object | string | null;
}

export interface CartCheckoutStateProps {
  step: number;
  products: CartProductStateProps[];
  subtotal: number;
  total: number;
  discount: number;
  shipping: number;
  billing: Address | null;
  payment: CartPaymentStateProps;
}

export interface CartProductStateProps {
  itemId?: string | number;
  id: string | number;
  name: string;
  image: string;
  salePrice: number;
  offerPrice: number;
  color: string;
  size: string | number;
  quantity: number;
  description?: string;
}

export type Address = {
  id?: string | number | undefined;
  name: string;
  destination: string;
  building: string;
  street: string;
  city: string;
  state: string;
  country: string;
  post: string | number;
  phone: string | number;
  isDefault: boolean;
};

export interface CartPaymentStateProps {
  type: string;
  method: string;
  card: string;
}

export interface ProductCardProps extends KeyedObject {
  id?: string | number;
  color?: string;
  name: string;
  image: string;
  description?: string;
  offerPrice?: number;
  salePrice?: number;
  rating?: number;
  open?: boolean;
}

export interface DefaultRootStateProps {
  cart: CartStateProps;
}
