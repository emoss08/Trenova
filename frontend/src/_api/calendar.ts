import services from 'utils/mockAdapter';

// third-party
import { v4 as UIDV4 } from 'uuid';
import _ from 'lodash';

import { events } from 'data/calendar';

// ==============================|| MOCK SERVICES ||============================== //

services.onGet('/api/calendar/events').reply(200, { events });

services.onPost('/api/calendar/events/add').reply((request) => {
  try {
    const { allDay, description, color, textColor, end, start, title } = JSON.parse(request.data);
    const event = {
      id: UIDV4(),
      allDay,
      description,
      color,
      textColor,
      end,
      start,
      title
    };
    events.push(event);

    return [200, { event }];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/calendar/events/update').reply((request) => {
  try {
    const { eventId, update } = JSON.parse(request.data);
    let event = null;

    _.map(events, (_event) => {
      if (_event.id === eventId) {
        _.assign(_event, { ...update });
        event = _event;
      }

      return _event;
    });

    return [200, { event }];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/calendar/events/delete').reply((request) => {
  try {
    const { eventId } = JSON.parse(request.data);
    _.reject(events, { id: eventId });

    return [200, { eventId }];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});
