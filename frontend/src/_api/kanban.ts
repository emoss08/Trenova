// third-party
import { Chance } from 'chance';
import { add, set, sub } from 'date-fns';

// project imports
import services from 'utils/mockAdapter';

// types
import { KanbanColumn, KanbanComment, KanbanItem, KanbanProfile, KanbanUserStory } from 'types/kanban';

const chance = new Chance();

// user profile data
const profileIdsData = {
  profile1: 'profile-1',
  profile2: 'profile-2',
  profile3: 'profile-3'
};

const profilesData: KanbanProfile[] = [
  {
    id: profileIdsData.profile1,
    avatar: 'avatar-3.png',
    name: 'Barney Thea',
    time: '2 min ago'
  },
  {
    id: profileIdsData.profile2,
    avatar: 'avatar-1.png',
    name: 'Maddison Wilber',
    time: '1 day ago'
  },
  {
    id: profileIdsData.profile3,
    avatar: 'avatar-2.png',
    name: 'John Doe',
    time: 'now'
  }
];

// task comment data
const commentIdsData = {
  comment1: 'comment-1',
  comment2: 'comment-2',
  comment3: 'comment-3',
  comment4: 'comment-4',
  comment5: 'comment-5'
};

const commentsData: KanbanComment[] = [
  {
    id: commentIdsData.comment1,
    comment: 'Comment 1',
    profileId: profileIdsData.profile1
  },
  {
    id: commentIdsData.comment2,
    comment: 'Comment 2',
    profileId: profileIdsData.profile2
  },
  {
    id: commentIdsData.comment3,
    comment: 'Comment 3',
    profileId: profileIdsData.profile3
  },
  {
    id: commentIdsData.comment4,
    comment: 'Comment 4',
    profileId: profileIdsData.profile2
  },
  {
    id: commentIdsData.comment5,
    comment: 'Comment 5',
    profileId: profileIdsData.profile3
  }
];

// items data
const itemIdsData = {
  item1: `${chance.integer({ min: 1000, max: 9999 })}`,
  item2: `${chance.integer({ min: 1000, max: 9999 })}`,
  item3: `${chance.integer({ min: 1000, max: 9999 })}`,
  item4: `${chance.integer({ min: 1000, max: 9999 })}`,
  item5: `${chance.integer({ min: 1000, max: 9999 })}`,
  item6: `${chance.integer({ min: 1000, max: 9999 })}`,
  item7: `${chance.integer({ min: 1000, max: 9999 })}`,
  item8: `${chance.integer({ min: 1000, max: 9999 })}`,
  item9: `${chance.integer({ min: 1000, max: 9999 })}`,
  item10: `${chance.integer({ min: 1000, max: 9999 })}`
};

const itemsData: KanbanItem[] = [
  {
    assign: profileIdsData.profile1,
    attachments: [],
    commentIds: [commentIdsData.comment1],
    description: 'Content of item 1',
    dueDate: sub(new Date(), { days: 12 }),
    id: itemIdsData.item1,
    image: 'profile-back-1.png',
    priority: 'low',
    title: 'Online fees payment & instant announcements'
  },
  {
    assign: profileIdsData.profile2,
    attachments: [],
    commentIds: [commentIdsData.comment2, commentIdsData.comment5],
    description: 'Content of item 2',
    dueDate: sub(new Date(), { days: 18 }),
    id: itemIdsData.item2,
    image: false,
    priority: 'high',
    title: 'Creation and Maintenance of Inventory Objects'
  },
  {
    assign: profileIdsData.profile3,
    attachments: [],
    description: 'Content of item 3',
    dueDate: sub(new Date(), { days: 8 }),
    id: itemIdsData.item3,
    image: false,
    priority: 'low',
    title: 'Update React & TypeScript version'
  },
  {
    assign: profileIdsData.profile2,
    attachments: [],
    commentIds: [commentIdsData.comment4],
    description: 'Content of item 4',
    dueDate: sub(new Date(), { days: 6 }),
    id: itemIdsData.item4,
    image: 'profile-back-2.png',
    priority: 'low',
    title: 'Set allowing rules for trusted applications.'
  },
  {
    assign: profileIdsData.profile2,
    attachments: [],
    commentIds: [commentIdsData.comment1, commentIdsData.comment2, commentIdsData.comment5],
    description: 'Content of item 5',
    dueDate: sub(new Date(), { days: 9 }),
    id: itemIdsData.item5,
    image: 'profile-back-3.png',
    priority: 'medium',
    title: 'Managing Applications Launch Control'
  },
  {
    assign: profileIdsData.profile3,
    attachments: [],
    commentIds: [commentIdsData.comment3, commentIdsData.comment4],
    description: 'Content of item 6',
    dueDate: set(new Date(), { hours: 10, minutes: 30 }),
    id: itemIdsData.item6,
    image: false,
    priority: 'medium',
    title: 'Run codemods'
  },
  {
    assign: profileIdsData.profile1,
    attachments: [],
    description: 'Content of item 7',
    dueDate: add(new Date(), { days: 5 }),
    id: itemIdsData.item7,
    image: 'profile-back-4.png',
    priority: 'low',
    title: 'Purchase Requisitions, Adjustments, and Transfers.'
  },
  {
    assign: profileIdsData.profile1,
    attachments: [],
    description: 'Content of item 8',
    dueDate: add(new Date(), { days: 17 }),
    id: itemIdsData.item8,
    image: false,
    priority: 'low',
    title: 'Attendance checking & homework details'
  },
  {
    assign: profileIdsData.profile3,
    attachments: [],
    commentIds: [commentIdsData.comment3],
    description: 'Content of item 9',
    dueDate: add(new Date(), { days: 8 }),
    id: itemIdsData.item9,
    image: false,
    priority: 'high',
    title: 'Admission, Staff & Schedule management'
  },
  {
    assign: profileIdsData.profile2,
    attachments: [],
    commentIds: [commentIdsData.comment5],
    description: 'Content of item 10',
    dueDate: add(new Date(), { days: 12 }),
    id: itemIdsData.item10,
    image: false,
    priority: 'low',
    title: 'Handling breaking changes'
  }
];

// columns data
const columnIdsData = {
  column1: 'column-1',
  column2: 'column-2',
  column3: 'column-3'
};

const columnsData: KanbanColumn[] = [
  {
    id: columnIdsData.column1,
    title: 'New',
    itemIds: [itemIdsData.item1, itemIdsData.item10, itemIdsData.item2]
  },
  {
    id: columnIdsData.column2,
    title: 'Active',
    itemIds: [itemIdsData.item8, itemIdsData.item5, itemIdsData.item4]
  },
  {
    id: columnIdsData.column3,
    title: 'Closed',
    itemIds: [itemIdsData.item3, itemIdsData.item9, itemIdsData.item7, itemIdsData.item6]
  }
];

const columnsOrderData: string[] = [columnIdsData.column1, columnIdsData.column2, columnIdsData.column3];

// user story data
const userStoryIdsData = {
  userStory1: `${chance.integer({ min: 1000, max: 9999 })}`,
  userStory2: `${chance.integer({ min: 1000, max: 9999 })}`,
  userStory3: `${chance.integer({ min: 1000, max: 9999 })}`,
  userStory4: `${chance.integer({ min: 1000, max: 9999 })}`
};

const userStoryOrderData: string[] = [
  userStoryIdsData.userStory1,
  userStoryIdsData.userStory2,
  userStoryIdsData.userStory3,
  userStoryIdsData.userStory4
];

const userStoryData: KanbanUserStory[] = [
  {
    acceptance: '',
    assign: profileIdsData.profile2,
    columnId: columnIdsData.column2,
    commentIds: [commentIdsData.comment5],
    description: chance.sentence(),
    dueDate: add(new Date(), { days: 12 }),
    id: userStoryIdsData.userStory1,
    priority: 'low',
    title: 'School Management Backend',
    itemIds: [itemIdsData.item1, itemIdsData.item8, itemIdsData.item9, itemIdsData.item7]
  },
  {
    acceptance: chance.sentence(),
    assign: profileIdsData.profile3,
    columnId: columnIdsData.column1,
    commentIds: [commentIdsData.comment3],
    description: chance.sentence(),
    dueDate: add(new Date(), { days: 8 }),
    id: userStoryIdsData.userStory2,
    priority: 'high',
    title: 'Inventory Implementation & Design',
    itemIds: [itemIdsData.item2, itemIdsData.item10]
  },
  {
    acceptance: chance.sentence({ words: 10 }),
    assign: profileIdsData.profile3,
    columnId: columnIdsData.column3,
    commentIds: [commentIdsData.comment3, commentIdsData.comment4],
    description: chance.sentence(),
    dueDate: set(new Date(), { hours: 10, minutes: 30 }),
    id: userStoryIdsData.userStory3,
    priority: 'medium',
    title: 'Theme migration from v4 to v5',
    itemIds: [itemIdsData.item3, itemIdsData.item6]
  },
  {
    acceptance: chance.sentence({ words: 5 }),
    assign: profileIdsData.profile1,
    columnId: columnIdsData.column2,
    commentIds: [commentIdsData.comment4],
    description: chance.sentence(),
    dueDate: sub(new Date(), { days: 8 }),
    id: userStoryIdsData.userStory4,
    priority: 'low',
    title: 'Lunch Beauty Application',
    itemIds: [itemIdsData.item4, itemIdsData.item5]
  }
];

// ==============================|| MOCK SERVICES ||============================== //

services.onGet('/api/kanban/columns').reply(200, { columns: columnsData });

services.onGet('/api/kanban/columns-order').reply(200, { columnsOrder: columnsOrderData });

services.onGet('/api/kanban/comments').reply(200, { comments: commentsData });

services.onGet('/api/kanban/profiles').reply(200, { profiles: profilesData });

services.onGet('/api/kanban/items').reply(200, { items: itemsData });

services.onGet('/api/kanban/userstory').reply(200, { userStory: userStoryData });

services.onGet('/api/kanban/userstory-order').reply(200, { userStoryOrder: userStoryOrderData });

services.onPost('/api/kanban/add-column').reply((config) => {
  try {
    const { column, columns, columnsOrder } = JSON.parse(config.data);
    const result = {
      columns: [...columns, column],
      columnsOrder: [...columnsOrder, column.id]
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/edit-column').reply((config) => {
  try {
    const { column, columns } = JSON.parse(config.data);

    columns.splice(
      columns.findIndex((c: KanbanColumn) => c.id === column.id),
      1,
      column
    );

    return [200, { columns }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/update-column-order').reply((config) => {
  try {
    const { columnsOrder } = JSON.parse(config.data);
    return [200, { columnsOrder }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/delete-column').reply((config) => {
  try {
    const { columnId, columnsOrder, columns } = JSON.parse(config.data);

    columns.splice(
      columns.findIndex((column: KanbanColumn) => column.id === columnId),
      1
    );

    columnsOrder.splice(
      columnsOrder.findIndex((cId: string) => cId === columnId),
      1
    );

    return [200, { columns, columnsOrder }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/add-item').reply((config) => {
  try {
    const { columnId, columns, item, items, storyId, userStory } = JSON.parse(config.data);
    let newColumn = columns;
    if (columnId !== '0') {
      newColumn = columns.map((column: KanbanColumn) => {
        if (column.id === columnId) {
          return {
            ...column,
            itemIds: column.itemIds ? [...column.itemIds, item.id] : [item.id]
          };
        }
        return column;
      });
    }

    let newUserStory = userStory;
    if (storyId !== '0') {
      newUserStory = userStory.map((story: KanbanUserStory) => {
        if (story.id === storyId) {
          return { ...story, itemIds: story.itemIds ? [...story.itemIds, item.id] : [item.id] };
        }
        return story;
      });
    }

    const result = {
      items: [...items, item],
      columns: newColumn,
      userStory: newUserStory
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/edit-item').reply((config) => {
  try {
    const { items, item, userStory, storyId, columns, columnId } = JSON.parse(config.data);
    items.splice(
      items.findIndex((i: KanbanItem) => i.id === item.id),
      1,
      item
    );

    let newUserStory = userStory;
    if (storyId) {
      const currentStory = userStory.filter((story: KanbanUserStory) => story.itemIds.filter((itemId: string) => itemId === item.id)[0])[0];
      if (currentStory !== undefined && currentStory.id !== storyId) {
        newUserStory = userStory.map((story: KanbanUserStory) => {
          if (story.itemIds.filter((itemId: string) => itemId === item.id)[0]) {
            return {
              ...story,
              itemIds: story.itemIds.filter((itemId: string) => itemId !== item.id)
            };
          }
          if (story.id === storyId) {
            return {
              ...story,
              itemIds: story.itemIds ? [...story.itemIds, item.id] : [item.id]
            };
          }
          return story;
        });
      }

      if (currentStory === undefined) {
        newUserStory = userStory.map((story: KanbanUserStory) => {
          if (story.id === storyId) {
            return {
              ...story,
              itemIds: story.itemIds ? [...story.itemIds, item.id] : [item.id]
            };
          }
          return story;
        });
      }
    }

    let newColumn = columns;
    if (columnId) {
      const currentColumn = columns.filter((column: KanbanColumn) => column.itemIds.filter((itemId: string) => itemId === item.id)[0])[0];
      if (currentColumn !== undefined && currentColumn.id !== columnId) {
        newColumn = columns.map((column: KanbanColumn) => {
          if (column.itemIds.filter((itemId: string) => itemId === item.id)[0]) {
            return {
              ...column,
              itemIds: column.itemIds.filter((itemId: string) => itemId !== item.id)
            };
          }
          if (column.id === columnId) {
            return {
              ...column,
              itemIds: column.itemIds ? [...column.itemIds, item.id] : [item.id]
            };
          }
          return column;
        });
      }

      if (currentColumn === undefined) {
        newColumn = columns.map((column: KanbanColumn) => {
          if (column.id === columnId) {
            return {
              ...column,
              itemIds: column.itemIds ? [...column.itemIds, item.id] : [item.id]
            };
          }
          return column;
        });
      }
    }

    const result = {
      items,
      columns: newColumn,
      userStory: newUserStory
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/update-item-order').reply((config) => {
  try {
    const { columns } = JSON.parse(config.data);
    return [200, { columns }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/select-item').reply((config) => {
  try {
    const { selectedItem } = JSON.parse(config.data);
    return [200, { selectedItem }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/add-item-comment').reply((config) => {
  try {
    const { items, itemId, comment, comments } = JSON.parse(config.data);

    const newItems = items.map((item: KanbanItem) => {
      if (item.id === itemId) {
        return {
          ...item,
          commentIds: item.commentIds ? [...item.commentIds, comment.id] : [comment.id]
        };
      }
      return item;
    });

    const result = {
      items: newItems,
      comments: [...comments, comment]
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/delete-item').reply((config) => {
  try {
    const { columns, itemId, userStory, items } = JSON.parse(config.data);

    const newColumn = columns.map((column: KanbanColumn) => {
      const itemIds = column.itemIds.filter((id: string) => id !== itemId);
      return {
        ...column,
        itemIds
      };
    });

    const newUserStory = userStory.map((story: KanbanUserStory) => {
      const itemIds = story.itemIds.filter((id: string) => id !== itemId);
      return {
        ...story,
        itemIds
      };
    });

    items.splice(
      items.findIndex((item: KanbanItem) => item.id === itemId),
      1
    );

    const result = {
      items,
      columns: newColumn,
      userStory: newUserStory
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/add-story').reply((config) => {
  try {
    const { userStory, story, userStoryOrder } = JSON.parse(config.data);

    const result = {
      userStory: [...userStory, story],
      userStoryOrder: [...userStoryOrder, story.id]
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/edit-story').reply((config) => {
  try {
    const { userStory, story } = JSON.parse(config.data);

    userStory.splice(
      userStory.findIndex((s: KanbanUserStory) => s.id === story.id),
      1,
      story
    );

    const result = {
      userStory
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/update-story-order').reply((config) => {
  try {
    const { userStoryOrder } = JSON.parse(config.data);
    return [200, { userStoryOrder }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/update-storyitem-order').reply((config) => {
  try {
    const { userStory } = JSON.parse(config.data);
    return [200, { userStory }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/add-story-comment').reply((config) => {
  try {
    const { userStory, storyId, comment, comments } = JSON.parse(config.data);

    const newUserStory = userStory.map((story: KanbanUserStory) => {
      if (story.id === storyId) {
        return {
          ...story,
          commentIds: story.commentIds ? [...story.commentIds, comment.id] : [comment.id]
        };
      }
      return story;
    });

    const result = {
      userStory: newUserStory,
      comments: [...comments, comment]
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});

services.onPost('/api/kanban/delete-story').reply((config) => {
  try {
    const { userStory, storyId, userStoryOrder } = JSON.parse(config.data);

    userStory.splice(
      userStory.findIndex((story: KanbanUserStory) => story.id === storyId),
      1
    );

    userStoryOrder.splice(
      userStoryOrder.findIndex((s: string) => s === storyId),
      1
    );

    const result = {
      userStory,
      userStoryOrder
    };

    return [200, { ...result }];
  } catch (err) {
    return [500, { message: 'Internal server error' }];
  }
});
