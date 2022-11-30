// ####################
// THESE ARE USED FOR TESTING THE GRAPHS
// ####################

const position = {x: 0, y: 0}

export const nodes = [
  {
    id: '1',
    type: 'input',
    data: {
      label: 'Input Node',
    },
    position,
  },
  {
    id: '2',
    data: {
      label: 'Default Node',
    },
    position,
  },
  {
    id: '3',
    type: 'output',
    data: {
      label: 'Output Node',
    },
    position,
  }
]

export const edges = [
    { id: 'e1-2', source: '1', target: '2', label: 'this is an edge label' },
    { id: 'e1-3', source: '1', target: '3', animated: true }
]