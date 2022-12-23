// third-party
import { add, sub } from 'date-fns';
import { Chance } from 'chance';

// types
import { Address, Products, Reviews } from 'types/e-commerce';

const chance = new Chance();

// product reviews list
export const productReviews: Reviews[] = [
  {
    id: '1',
    rating: 3.5,
    review: chance.paragraph({ sentences: 2 }),
    date: '2 hour ago',
    profile: {
      avatar: 'avatar-1.png',
      name: 'Emma Labelle',
      status: chance.bool()
    }
  },
  {
    id: '2',
    rating: 4.0,
    review: chance.paragraph({ sentences: 2 }),
    date: '12 hour ago',
    profile: {
      avatar: 'avatar-2.png',
      name: 'Lucifer Wing',
      status: chance.bool()
    }
  },
  {
    id: '3',
    rating: 4.5,
    review: 'Nice!',
    date: '1 day ago',
    profile: {
      avatar: 'avatar-3.png',
      name: 'John smith',
      status: chance.bool()
    }
  }
];

// billing address list
export const address: Address[] = [
  {
    id: 1,
    name: 'Ian Carpenter',
    destination: 'home',
    building: '1754 Ureate Path',
    street: '695 Newga View',
    city: 'Seporcus',
    state: 'Rhode Island',
    country: 'Belgium',
    post: 'SA5 5BO',
    phone: '+91 1234567890',
    isDefault: true
  },
  {
    id: 2,
    name: 'Ian Carpenter',
    destination: 'office',
    building: '1754 Ureate Path',
    street: '695 Newga View',
    city: 'Seporcus',
    state: 'Rhode Island',
    country: 'Belgium',
    post: 'SA5 5BO',
    phone: '+91 1234567890',
    isDefault: false
  }
];

// products list
export const products: Products[] = [
  {
    id: 1,
    image: 'prod-11.png',
    name: 'Apple Series 4 GPS A38 MM Space',
    brand: 'Apple',
    description: 'Apple Watch SE Smartwatch ',
    about:
      'This watch from Apple is highly known for its features, like you can control your apple smartphone with this watch and you can do everything you would want to do on smartphone',
    quantity: 3,
    rating: 4.0,
    discount: 75,
    offerPrice: 275,
    gender: 'male',
    categories: ['fashion', 'watch'],
    colors: ['errorDark', 'errorMain', 'secondaryMain'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 8, hours: 6, minutes: 20 }),
    isStock: true,
    new: 45
  },
  {
    id: 2,
    image: 'prod-22.png',
    name: 'Boat On-Ear Wireless ',
    brand: 'Boat',
    description: 'Mic(Bluetooth 4.2, Rockerz 450R...',
    about:
      'Boat On-ear wireless headphones comes with bluethooth technology, comes with better sound quality and weighs around 200gm which seems very light when using ',
    quantity: 45,
    rating: 3.5,
    discount: 10,
    offerPrice: 81.99,
    gender: 'kids',
    categories: ['electronics', 'headphones'],
    colors: ['primary200', 'successLight', 'secondary200', 'warningMain'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 10, hours: 8, minutes: 69 }),
    isStock: false,
    new: 40
  },
  {
    id: 3,
    image: 'prod-33.png',
    name: 'Fitbit MX30 Smart Watch',
    brand: 'Fitbit',
    offer: '30%',
    description: '(MX30- waterproof) watch',
    about:
      'Fitbit is well known for making amazing smartwatches and this product is one of them, it is waterproof and battery power can last upto 2 days. Great product for smartwatch lovers',
    quantity: 70,
    rating: 4.5,
    discount: 40,
    salePrice: 85.0,
    offerPrice: 49.9,
    gender: 'male',
    categories: ['fashion', 'watch'],
    colors: ['primary200', 'primaryDark'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 4, hours: 9, minutes: 50 }),
    isStock: true,
    new: 35
  },
  {
    id: 4,
    image: 'prod-44.png',
    name: 'Luxury Watches Centrix Gold',
    brand: 'Centrix',
    offer: '30%',
    description: '7655 Couple (Refurbished)...',
    about: 'This product from Centrix is a classic choice who love classical products. It it made of pure gold and weighs around 50 grams',
    quantity: 3,
    rating: 4.0,
    discount: 20,
    salePrice: 36.0,
    offerPrice: 29.99,
    gender: 'kids',
    categories: ['fashion', 'watch'],
    colors: ['errorLight', 'warningMain'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 7, hours: 6, minutes: 45 }),
    isStock: true,
    new: 15
  },
  {
    id: 5,
    image: 'prod-55.png',
    name: 'Canon EOS 1500D 24.1 Digital SLR',
    brand: 'Canon',
    offer: '30%',
    description: 'SLR Camera (Black) with EF S18-55...',
    about:
      'Image Enlargement: After shooting, you can enlarge photographs of the objects for clear zoomed view. Change In Aspect Ratio: Boldly crop the subject and save it with a composition that has impact. You can convert it to a 1:1 square format, and after shooting you can create a photo that will be popular on SNS.',
    quantity: 50,
    rating: 3.5,
    discount: 15,
    salePrice: 15.99,
    offerPrice: 12.99,
    gender: 'male',
    categories: ['electronics', 'camera'],
    colors: ['warningMain', 'primary200'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 2, hours: 9, minutes: 45 }),
    isStock: true,
    new: 25
  },
  {
    id: 6,
    image: 'prod-66.png',
    name: 'Apple iPhone 13 Mini ',
    brand: 'Apple',
    offer: '30%',
    description: '13 cm (5.4-inch) Super',
    about:
      'It fits for those who love photography since it has 48MP camera which shoots great photos even in low light. Also it has 8GB of RAm and 4000MAH battery which can last upto 12 hours a day ',
    quantity: 40,
    rating: 4.5,
    discount: 10,
    offerPrice: 86.99,
    gender: 'female',
    categories: ['electronics', 'iphone'],
    colors: ['primaryDark'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: add(new Date(), { days: 6, hours: 10, minutes: 0 }),
    isStock: true,
    new: 15
  },
  {
    id: 7,
    image: 'prod-77.png',
    name: 'Apple MacBook Pro with Iphone',
    brand: 'Apple',
    description: '11th Generation Intel® Core™ i5-11320H ...',
    about:
      'Great choice for those who love poweful and fast laptopts. It comes with 2TB of harddrive and 12GB of RAM.Its fast and comes with a powerful processor',
    quantity: 70,
    rating: 4.0,
    discount: 16,
    offerPrice: 14.59,
    gender: 'male',
    categories: ['electronics', 'laptop'],
    colors: ['errorDark', 'secondaryMain', 'errorMain'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: add(new Date(), { days: 14, hours: 1, minutes: 55 }),
    isStock: true,
    new: 30
  },
  {
    id: 8,
    image: 'prod-88.png',
    name: 'Apple iPhone 13 Pro',
    brand: 'Apple',
    description: '(512GB ROM, MLLH3HN/A,..',
    about:
      'The smartphone comes with 12GB of RAM and 2Ghz of processor.There are two front cameras, one of 20MP and second of 10MP for depth phptpgraphy.Its lightweight and fast',
    quantity: 21,
    rating: 4.5,
    discount: 30,
    salePrice: 129.99,
    offerPrice: 100.0,
    gender: 'female',
    categories: ['electronics', 'iphone'],
    colors: ['errorMain', 'successDark'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 0, hours: 11, minutes: 10 }),
    isStock: true,
    new: 20
  },

  {
    id: 9,
    image: 'prod-99.png',
    name: 'Canon EOS 1500D 24.1 Digital',
    brand: 'Kotak',
    description: '(512GB ROM, MLLH3HN/A,..',
    about:
      'Image Enlargement: After shooting, you can enlarge photographs of the objects for clear zoomed view. Change In Aspect Ratio: Boldly crop the subject and save it with a composition that has impact. You can convert it to a 1:1 square format, and after shooting you can create a photo that will be popular on SNS.',
    quantity: 21,
    rating: 4.0,
    discount: 5,
    offerPrice: 399,
    gender: 'female',
    categories: ['electronics', 'camera'],
    colors: ['errorMain', 'successDark'],
    popularity: chance.natural(),
    date: chance.natural(),
    created: sub(new Date(), { days: 0, hours: 11, minutes: 10 }),
    isStock: true,
    new: 10
  }
];
