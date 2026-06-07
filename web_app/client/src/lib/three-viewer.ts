import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { computePointBounds, normalizePoints, pointBoundsScale } from "./mesh-parser";
import type { MeshData, MeshFrame, PointBounds } from "../types";

export type RenderOptions = {
  jobId?: string | null;
  frames?: MeshFrame[];
};

export class ThreeMeshViewer {
  private readonly container: HTMLElement;
  private readonly scene: THREE.Scene;
  private readonly camera: THREE.PerspectiveCamera;
  private readonly renderer: THREE.WebGLRenderer;
  private readonly controls: OrbitControls;
  private meshGroup: THREE.Group | null = null;
  private viewportJobId: string | null = null;
  private viewportBounds: PointBounds | null = null;
  private viewportCameraFit = false;
  private resizeObserver: ResizeObserver;
  private animationId: number | null = null;

  constructor(container: HTMLElement) {
    this.container = container;
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(0x101820);

    this.camera = new THREE.PerspectiveCamera(45, 1, 0.01, 1000);
    this.renderer = new THREE.WebGLRenderer({ antialias: true });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    this.container.appendChild(this.renderer.domElement);

    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.08;

    this.scene.add(new THREE.AmbientLight(0xffffff, 0.6));
    const keyLight = new THREE.DirectionalLight(0xffffff, 1.1);
    keyLight.position.set(4, 6, 8);
    this.scene.add(keyLight);

    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(this.container);
    this.resize();
    this.animate();
  }

  renderPointCloud(pointCloud: MeshData, options: RenderOptions = {}): void {
    const viewport = this.updateViewport(pointCloud, options);
    if (this.meshGroup) {
      this.scene.remove(this.meshGroup);
      disposeObject(this.meshGroup);
    }

    const normalized = normalizePoints(pointCloud.points, viewport.bounds);
    const group = new THREE.Group();
    const mobilePositions: number[] = [];
    const fixedPositions: number[] = [];

    for (const point of normalized.points) {
      const target = point.fixed ? fixedPositions : mobilePositions;
      target.push(point.x, point.y, point.z);
    }

    if (mobilePositions.length > 0) {
      group.add(createPointLayer(mobilePositions, 0x55a7ff, 0.22));
    }
    if (fixedPositions.length > 0) {
      group.add(createPointLayer(fixedPositions, 0xffc857, 0.34));
    }

    const edges = pointCloud.edges || [];
    if (edges.length > 0) {
      const edgePositions: number[] = [];
      for (const edge of edges) {
        const p1 = normalized.points[edge.a];
        const p2 = normalized.points[edge.b];
        if (!p1 || !p2) continue;
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

    const zeroPlane = new THREE.GridHelper(Math.max(normalized.span, 1), 16, 0x4b6f91, 0x263b4d);
    zeroPlane.position.y = 0;
    const zeroPlaneMaterials = Array.isArray(zeroPlane.material) ? zeroPlane.material : [zeroPlane.material];
    for (const material of zeroPlaneMaterials) {
      material.opacity = 0.45;
      material.transparent = true;
    }
    group.add(zeroPlane);
    group.add(new THREE.AxesHelper(Math.max(normalized.span, 1) * 0.35));

    this.scene.add(group);
    this.meshGroup = group;
    if (viewport.shouldFitCamera) {
      this.fitCameraToBox(createNormalizedBoxFromBounds(viewport.bounds));
      this.viewportCameraFit = true;
    }
  }

  clear(jobId: string | null = null): void {
    if (this.meshGroup) {
      this.scene.remove(this.meshGroup);
      disposeObject(this.meshGroup);
      this.meshGroup = null;
    }
    this.viewportJobId = jobId;
    this.viewportBounds = null;
    this.viewportCameraFit = false;
  }

  dispose(): void {
    if (this.animationId !== null) {
      window.cancelAnimationFrame(this.animationId);
      this.animationId = null;
    }
    this.resizeObserver.disconnect();
    this.clear();
    this.controls.dispose();
    this.renderer.dispose();
    this.renderer.domElement.remove();
  }

  private updateViewport(pointCloud: MeshData, options: RenderOptions): { bounds: PointBounds; shouldFitCamera: boolean } {
    const jobId = options.jobId || null;
    const bounds = computeLoadedFrameBounds(pointCloud, options.frames || []);
    const switchedJobs = this.viewportJobId !== jobId;
    const expandedBounds = !switchedJobs && boundsExceed(this.viewportBounds, bounds);

    if (switchedJobs) {
      this.viewportJobId = jobId;
      this.viewportBounds = bounds;
      this.viewportCameraFit = false;
    } else if (expandedBounds) {
      this.viewportBounds = mergeBounds(this.viewportBounds, bounds);
      this.viewportCameraFit = false;
    } else if (!this.viewportBounds) {
      this.viewportBounds = bounds;
    }

    return {
      bounds: this.viewportBounds,
      shouldFitCamera: !this.viewportCameraFit,
    };
  }

  private fitCameraToBox(box: THREE.Box3): void {
    const size = new THREE.Vector3();
    const origin = new THREE.Vector3(0, 0, 0);
    box.getSize(size);

    const maxDim = Math.max(size.x, size.y, size.z, 1);
    const fov = this.camera.fov * (Math.PI / 180);
    const distance = Math.abs(maxDim / (2 * Math.tan(fov / 2))) * 1.8;

    this.camera.position.set(
      distance * 0.65,
      distance * 0.4,
      distance,
    );
    this.camera.near = Math.max(distance / 1000, 0.01);
    this.camera.far = distance * 100;
    this.camera.updateProjectionMatrix();
    this.controls.target.copy(origin);
    this.controls.update();
  }

  private resize(): void {
    const width = this.container.clientWidth;
    const height = this.container.clientHeight;
    if (width <= 0 || height <= 0) return;

    this.camera.aspect = width / height;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(width, height, false);
  }

  private animate = (): void => {
    this.animationId = requestAnimationFrame(this.animate);
    this.controls.update();
    this.renderer.render(this.scene, this.camera);
  };
}

function createPointLayer(positions: number[], color: number, size: number): THREE.Points {
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
  const material = new THREE.PointsMaterial({
    color,
    size,
    sizeAttenuation: true,
  });
  return new THREE.Points(geometry, material);
}

function computeLoadedFrameBounds(pointCloud: MeshData, frames: MeshFrame[]): PointBounds {
  const pointSets = [pointCloud.points];
  for (const frame of frames) {
    if (frame.loaded && frame.pointCloud?.points) {
      pointSets.push(frame.pointCloud.points);
    }
  }
  return computePointBounds(pointSets) || {
    minX: 0,
    minY: 0,
    minZ: 0,
    maxX: 0,
    maxY: 0,
    maxZ: 0,
  };
}

function boundsExceed(current: PointBounds | null, next: PointBounds): boolean {
  if (!current) return true;
  const epsilon = 1e-9;
  return next.minX < current.minX - epsilon
    || next.minY < current.minY - epsilon
    || next.minZ < current.minZ - epsilon
    || next.maxX > current.maxX + epsilon
    || next.maxY > current.maxY + epsilon
    || next.maxZ > current.maxZ + epsilon;
}

function mergeBounds(current: PointBounds | null, next: PointBounds): PointBounds {
  if (!current) return next;
  return {
    minX: Math.min(current.minX, next.minX),
    minY: Math.min(current.minY, next.minY),
    minZ: Math.min(current.minZ, next.minZ),
    maxX: Math.max(current.maxX, next.maxX),
    maxY: Math.max(current.maxY, next.maxY),
    maxZ: Math.max(current.maxZ, next.maxZ),
  };
}

function createNormalizedBoxFromBounds(bounds: PointBounds): THREE.Box3 {
  const scale = pointBoundsScale(bounds);
  return new THREE.Box3(
    new THREE.Vector3(
      Math.min(bounds.minX * scale, 0),
      Math.min(bounds.minY * scale, 0),
      Math.min(bounds.minZ * scale, 0),
    ),
    new THREE.Vector3(
      Math.max(bounds.maxX * scale, 0),
      Math.max(bounds.maxY * scale, 0),
      Math.max(bounds.maxZ * scale, 0),
    ),
  );
}

function disposeObject(object: THREE.Object3D): void {
  object.traverse((child) => {
    const mesh = child as THREE.Object3D & {
      geometry?: { dispose: () => void };
      material?: { dispose: () => void } | Array<{ dispose: () => void }>;
    };
    if (mesh.geometry) mesh.geometry.dispose();
    if (mesh.material) {
      const materials = Array.isArray(mesh.material) ? mesh.material : [mesh.material];
      for (const material of materials) material.dispose();
    }
  });
}
