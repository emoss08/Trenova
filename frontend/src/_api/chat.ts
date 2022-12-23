// project imports
import services from 'utils/mockAdapter';

// types
import { ChatHistory } from 'types/chat';

import { users, text1, text2, text3, text4, text5, text6, text7, text8 } from 'data/chat';

// chat history
const chatHistories: ChatHistory[] = [
  { id: 1, from: 'User1', to: 'Alene', text: text1, time: '11:23 AM' },
  { id: 2, from: 'Alene', to: 'User1', text: text2, time: '11:23 AM' },
  { id: 3, from: 'User1', to: 'Alene', text: text3, time: '11:23 AM' },
  { id: 4, from: 'Alene', to: 'User1', text: text4, time: '11:23 AM' },

  { id: 5, from: 'User1', to: 'Keefe', text: text5, time: '11:24 AM' },
  { id: 6, from: 'Keefe', to: 'User1', text: text6, time: '11:24 AM' },
  { id: 7, from: 'User1', to: 'Keefe', text: text7, time: '11:24 AM' },
  { id: 8, from: 'Keefe', to: 'User1', text: text8, time: '11:24 AM' },

  { id: 9, from: 'User1', to: 'Lazaro', text: text1, time: '11:25 AM' },
  { id: 10, from: 'Lazaro', to: 'User1', text: text2, time: '11:25 AM' },
  { id: 11, from: 'User1', to: 'Lazaro', text: text3, time: '11:25 AM' },
  { id: 12, from: 'Lazaro', to: 'User1', text: text4, time: '11:25 AM' },

  { id: 13, from: 'User1', to: 'Hazle', text: text5, time: '11:26 AM' },
  { id: 14, from: 'Hazle', to: 'User1', text: text6, time: '11:26 AM' },
  { id: 15, from: 'User1', to: 'Hazle', text: text7, time: '11:26 AM' },
  { id: 16, from: 'Hazle', to: 'User1', text: text8, time: '11:26 AM' },

  { id: 17, from: 'User1', to: 'Herman Essertg', text: text1, time: '11:27 AM' },
  { id: 18, from: 'Herman Essertg', to: 'User1', text: text2, time: '11:27 AM' },
  { id: 19, from: 'User1', to: 'Herman Essertg', text: text3, time: '11:27 AM' },
  { id: 20, from: 'Herman Essertg', to: 'User1', text: text4, time: '11:27 AM' },

  { id: 21, from: 'User1', to: 'Wilhelmine Durrg', text: text5, time: '11:28 AM' },
  { id: 22, from: 'Wilhelmine Durrg', to: 'User1', text: text6, time: '11:28 AM' },
  { id: 23, from: 'User1', to: 'Wilhelmine Durrg', text: text7, time: '11:28 AM' },
  { id: 24, from: 'Wilhelmine Durrg', to: 'User1', text: text8, time: '11:28 AM' },

  { id: 25, from: 'User1', to: 'Agilulf Fuxg', text: text1, time: '11:29 AM' },
  { id: 26, from: 'Agilulf Fuxg', to: 'User1', text: text2, time: '11:29 AM' },
  { id: 27, from: 'User1', to: 'Agilulf Fuxg', text: text3, time: '11:29 AM' },
  { id: 28, from: 'Agilulf Fuxg', to: 'User1', text: text4, time: '11:29 AM' },

  { id: 29, from: 'User1', to: 'Adaline Bergfalks', text: text5, time: '11:30 AM' },
  { id: 30, from: 'Adaline Bergfalks', to: 'User1', text: text6, time: '11:30 AM' },
  { id: 31, from: 'User1', to: 'Adaline Bergfalks', text: text7, time: '11:30 AM' },
  { id: 32, from: 'Adaline Bergfalks', to: 'User1', text: text8, time: '11:30 AM' },

  { id: 33, from: 'User1', to: 'Eadwulf Beckete', text: text1, time: '11:31 AM' },
  { id: 34, from: 'Eadwulf Beckete', to: 'User1', text: text2, time: '11:31 AM' },
  { id: 35, from: 'User1', to: 'Eadwulf Beckete', text: text3, time: '11:31 AM' },
  { id: 36, from: 'Eadwulf Beckete', to: 'User1', text: text4, time: '11:31 AM' },

  { id: 37, from: 'User1', to: 'Midas', text: text5, time: '11:32 AM' },
  { id: 38, from: 'Midas', to: 'User1', text: text6, time: '11:32 AM' },
  { id: 39, from: 'User1', to: 'Midas', text: text7, time: '11:32 AM' },
  { id: 40, from: 'Midas', to: 'User1', text: text8, time: '11:32 AM' },

  { id: 41, from: 'User1', to: 'Uranus', text: text1, time: '11:33 AM' },
  { id: 42, from: 'Uranus', to: 'User1', text: text2, time: '11:33 AM' },
  { id: 43, from: 'User1', to: 'Uranus', text: text3, time: '11:33 AM' },
  { id: 44, from: 'Uranus', to: 'User1', text: text4, time: '11:33 AM' },

  { id: 45, from: 'User1', to: 'Peahen', text: text5, time: '11:34 AM' },
  { id: 46, from: 'Peahen', to: 'User1', text: text6, time: '11:34 AM' },
  { id: 47, from: 'User1', to: 'Peahen', text: text7, time: '11:34 AM' },
  { id: 48, from: 'Peahen', to: 'User1', text: text8, time: '11:34 AM' },

  { id: 49, from: 'User1', to: 'Menelaus', text: text1, time: '11:35 AM' },
  { id: 50, from: 'Menelaus', to: 'User1', text: text2, time: '11:35 AM' },
  { id: 51, from: 'User1', to: 'Menelaus', text: text3, time: '11:35 AM' },
  { id: 52, from: 'Menelaus', to: 'User1', text: text4, time: '11:35 AM' }
];

// ==============================|| MOCK SERVICES ||============================== //

services.onGet('/api/chat/users').reply(200, { users });

services.onPost('/api/chat/users/id').reply((config) => {
  try {
    const { id } = JSON.parse(config.data);
    const index = users.findIndex((x) => x.id === id);
    return [200, users[index]];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/chat/filter').reply(async (config) => {
  try {
    const { user } = JSON.parse(config.data);
    const result = chatHistories.filter((item) => item.from === user || item.to === user);
    return [200, result];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/chat/insert').reply((config) => {
  try {
    const { from, to, text, time } = JSON.parse(config.data);
    const result = chatHistories.push({ from, to, text, time });
    return [200, result];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/chat/users/modify').reply((config) => {
  try {
    const user = JSON.parse(config.data);
    if (user.id) {
      const index = users.findIndex((u) => u.id === user.id);
      if (index > -1) {
        users[index] = { ...users[index], ...user };
      }
    } else {
      users.push({ ...user, id: users.length + 1 });
    }
    return [200, []];
  } catch (err) {
    console.error(err);
    return [500, { message: 'Internal server error' }];
  }
});
