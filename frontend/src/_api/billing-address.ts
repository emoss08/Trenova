// project imports
import services from 'utils/mockAdapter';

// third-party
import { v4 as UIDV4 } from 'uuid';

import { address } from 'data/e-commerce';

// ==============================|| MOCK SERVICES ||============================== //

services.onGet('/api/address/list').reply(200, { address });

services.onPost('/api/address/new').reply((request) => {
  try {
    const data = JSON.parse(request.data);
    const { name, destination, building, street, city, state, country, post, phone, isDefault } = data;
    const newAddress = {
      id: UIDV4(),
      name,
      destination,
      building,
      street,
      city,
      state,
      country,
      post,
      phone,
      isDefault
    };

    let result = address;
    if (isDefault) {
      result = address.map((item) => {
        if (item.isDefault === true) {
          return { ...item, isDefault: false };
        }
        return item;
      });
    }

    result = [...address, newAddress];

    return [200, { address: result }];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/address/edit').reply((request) => {
  try {
    const data = JSON.parse(request.data);

    let result = address;
    if (data.isDefault) {
      result = address.map((item) => {
        if (item.isDefault === true) {
          return { ...item, isDefault: false };
        }
        return item;
      });
    }

    result = address.map((item) => {
      if (item.id === data.id) {
        return data;
      }
      return item;
    });

    return [200, { result }];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});
