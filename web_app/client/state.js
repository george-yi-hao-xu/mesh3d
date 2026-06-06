export const state = {
  currentUser: null,
  jobs: [],
  activeJobId: null,
  jobNameEdited: false,
  events: null,
  meshCache: new Map(),
  viewer: {
    ready: null,
    disabled: false,
    THREE: null,
    renderer: null,
    scene: null,
    camera: null,
    controls: null,
    meshGroup: null,
    resizeObserver: null,
    animationId: null,
  },
};

export const maxCachedMeshes = 80;
