/**
 * @typedef {{
 *   ready: Promise<ViewerState | null> | null,
 *   disabled: boolean,
 *   THREE: any,
 *   renderer: any,
 *   scene: any,
 *   camera: any,
 *   controls: any,
 *   meshGroup: any,
 *   resizeObserver: ResizeObserver | null,
 *   animationId: number | null
 * }} ViewerState
 *
 * @typedef {{
 *   currentUser: import("./api.js").User | null,
 *   jobs: import("./api.js").Job[],
 *   activeJobId: string | null,
 *   activeFrameUrl: string | null,
 *   activeFrames: import("./api.js").MeshFrame[],
 *   jobNameEdited: boolean,
 *   jobCache: Map<string, import("./api.js").Job>,
 *   viewer: ViewerState
 * }} ClientState
 */

/** @type {ClientState} */
export const state = {
  currentUser: null,
  jobs: [],
  activeJobId: null,
  activeFrameUrl: null,
  activeFrames: [],
  jobNameEdited: false,
  jobCache: new Map(),
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
