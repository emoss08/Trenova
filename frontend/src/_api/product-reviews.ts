// project imports
import services from 'utils/mockAdapter';

import { productReviews } from 'data/e-commerce';

// ==============================|| MOCK SERVICES ||============================== //

services.onGet('/api/review/list').reply(200, { productReviews });
