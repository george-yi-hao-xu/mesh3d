import { inferEdges, normalizePoints } from "./mesh-parser.js";

/**
 * @typedef {{
 *   meshCanvas: HTMLElement,
 *   meshCanvasMessage: HTMLElement
 * }} ViewerElements
 */

/**
 * Renders a parsed point cloud into the shared Three.js scene.
 *
 * @param {import("./state.js").ClientState} state
 * @param {ViewerElements} els
 * @param {import("./mesh-parser.js").PointCloud} pointCloud
 * @returns {Promise<void>}
 */
export async function renderPointCloud(state, els, pointCloud) {
  const viewer = await ensureViewer(state, els);
  if (!viewer) return;

  const { THREE, scene } = viewer;
  if (viewer.meshGroup) {
    scene.remove(viewer.meshGroup);
    disposeObject(viewer.meshGroup);
  }

  const normalized = normalizePoints(pointCloud.points);
  const group = new THREE.Group();
  const mobilePositions = [];
  const fixedPositions = [];

  for (const point of normalized.points) {
    const target = point.fixed ? fixedPositions : mobilePositions;
    target.push(point.x, point.y, point.z);
  }

  if (mobilePositions.length > 0) {
    group.add(createPointLayer(THREE, mobilePositions, 0x55a7ff, 0.22));
  }
  if (fixedPositions.length > 0) {
    group.add(createPointLayer(THREE, fixedPositions, 0xffc857, 0.34));
  }

  const edges = inferEdges(normalized.points);
  if (edges.length > 0) {
    const edgePositions = [];
    for (const [a, b] of edges) {
      const p1 = normalized.points[a];
      const p2 = normalized.points[b];
      edgePositions.push(p1.x, p1.y, p1.z, p2.x, p2.y, p2.z);
    }

    const edgeGeometry = new THREE.BufferGeometry();
    edgeGeometry.setAttribute("position", new THREE.Float32BufferAttribute(edgePositions, 3));
    const edgeMaterial = new THREE.LineBasicMaterial({
      color: 0x8fb3d9,
      transparent: true,
      opacity: 0.42,
    });
    group.add(new THREE.LineSegments(edgeGeometry, edgeMaterial));
  }

  const box = new THREE.Box3(
    new THREE.Vector3(normalized.min.x, normalized.min.y, normalized.min.z),
    new THREE.Vector3(normalized.max.x, normalized.max.y, normalized.max.z),
  );
  const zeroPlane = new THREE.GridHelper(Math.max(normalized.span, 1), 16, 0x4b6f91, 0x263b4d);
  zeroPlane.position.y = 0;
  const zeroPlaneMaterials = Array.isArray(zeroPlane.material) ? zeroPlane.material : [zeroPlane.material];
  for (const material of zeroPlaneMaterials) {
    material.opacity = 0.45;
    material.transparent = true;
  }
  group.add(zeroPlane);
  group.add(new THREE.Box3Helper(box, 0x3d5368));
  group.add(new THREE.AxesHelper(Math.max(normalized.span, 1) * 0.35));

  scene.add(group);
  viewer.meshGroup = group;
  fitCameraToBox(viewer, box);
  setMeshMessage(els, "", true);
}

/**
 * Updates the overlay message shown on top of the mesh canvas.
 *
 * @param {ViewerElements} els
 * @param {string} message
 * @param {boolean} [hidden]
 * @returns {void}
 */
export function setMeshMessage(els, message, hidden = false) {
  els.meshCanvasMessage.textContent = message;
  els.meshCanvasMessage.classList.toggle("hidden", hidden);
}

/**
 * Returns the existing viewer or lazily creates it on first render.
 *
 * @param {import("./state.js").ClientState} state
 * @param {ViewerElements} els
 * @returns {Promise<import("./state.js").ViewerState | null>}
 */
async function ensureViewer(state, els) {
  if (state.viewer.disabled) {
    setMeshMessage(els, "3D view unavailable. Raw mesh text is shown below.");
    return null;
  }

  if (!state.viewer.ready) {
    state.viewer.ready = setupViewer(state, els).catch((error) => {
      state.viewer.disabled = true;
      console.error(error);
      setMeshMessage(els, "Could not load Three.js. Raw mesh text is shown below.");
      return null;
    });
  }

  return state.viewer.ready;
}

/**
 * Creates the Three.js renderer, scene, camera, controls, lights, resize observer, and render loop.
 *
 * @param {import("./state.js").ClientState} state
 * @param {ViewerElements} els
 * @returns {Promise<import("./state.js").ViewerState>}
 */
async function setupViewer(state, els) {
  setMeshMessage(els, "Loading Three.js.");
  const [THREE, { OrbitControls }] = await Promise.all([
    import("three"),
    import("three/addons/controls/OrbitControls.js"),
  ]);

  const scene = new THREE.Scene();
  scene.background = new THREE.Color(0x101820);

  const camera = new THREE.PerspectiveCamera(45, 1, 0.01, 1000);
  const renderer = new THREE.WebGLRenderer({ antialias: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  els.meshCanvas.appendChild(renderer.domElement);

  const controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.dampingFactor = 0.08;

  scene.add(new THREE.AmbientLight(0xffffff, 0.6));
  const keyLight = new THREE.DirectionalLight(0xffffff, 1.1);
  keyLight.position.set(4, 6, 8);
  scene.add(keyLight);

  Object.assign(state.viewer, {
    THREE,
    renderer,
    scene,
    camera,
    controls,
  });

  const resize = () => resizeViewer(state.viewer, els);
  state.viewer.resizeObserver = new ResizeObserver(resize);
  state.viewer.resizeObserver.observe(els.meshCanvas);
  resize();

  const animate = () => {
    state.viewer.animationId = requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
  };
  animate();

  return state.viewer;
}

/**
 * Creates one point-cloud layer for either fixed or moving particles.
 *
 * @param {any} THREE
 * @param {number[]} positions
 * @param {number} color
 * @param {number} size
 * @returns {any}
 */
function createPointLayer(THREE, positions, color, size) {
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
  const material = new THREE.PointsMaterial({
    color,
    size,
    sizeAttenuation: true,
  });
  return new THREE.Points(geometry, material);
}

/**
 * Frames the scene while keeping orbit controls centered on the true solver origin.
 *
 * @param {import("./state.js").ViewerState} viewer
 * @param {any} box
 * @returns {void}
 */
function fitCameraToBox(viewer, box) {
  const { THREE, camera, controls } = viewer;
  const size = new THREE.Vector3();
  const origin = new THREE.Vector3(0, 0, 0);
  box.getSize(size);

  const maxDim = Math.max(size.x, size.y, size.z, 1);
  const fov = camera.fov * (Math.PI / 180);
  const distance = Math.abs(maxDim / (2 * Math.tan(fov / 2))) * 1.8;

  camera.position.set(
    distance * 0.65,
    distance * 0.4,
    distance,
  );
  camera.near = Math.max(distance / 1000, 0.01);
  camera.far = distance * 100;
  camera.updateProjectionMatrix();
  controls.target.copy(origin);
  controls.update();
}

/**
 * Resizes the WebGL renderer to the current canvas container dimensions.
 *
 * @param {import("./state.js").ViewerState} viewer
 * @param {ViewerElements} els
 * @returns {void}
 */
function resizeViewer(viewer, els) {
  if (!viewer.renderer || !viewer.camera) return;

  const width = els.meshCanvas.clientWidth;
  const height = els.meshCanvas.clientHeight;
  if (width <= 0 || height <= 0) return;

  viewer.camera.aspect = width / height;
  viewer.camera.updateProjectionMatrix();
  viewer.renderer.setSize(width, height, false);
}

/**
 * Disposes geometries and materials before replacing the previous mesh group.
 *
 * @param {any} object
 * @returns {void}
 */
function disposeObject(object) {
  object.traverse((child) => {
    if (child.geometry) child.geometry.dispose();
    if (child.material) {
      const materials = Array.isArray(child.material) ? child.material : [child.material];
      for (const material of materials) material.dispose();
    }
  });
}
