export type User = {
  id: string;
  username: string;
};

export type Upload = {
  id: string;
  userId?: string;
  fileName: string;
  size: number;
  meshKind?: "uploaded" | "generated";
  pointCount?: number;
  edgeCount?: number;
  createdAt: string;
};

export type UploadArtifact = {
  upload: Upload;
  text: string;
};

export type Snapshot = {
  label: string;
  simTime?: number;
  step?: number;
  url: string;
  createdAt?: string;
};

export type JobReview = {
  jobId: string;
  score: number;
  tags: string[];
  note?: string;
  createdAt: string;
  updatedAt: string;
};

export type Job = {
  id: string;
  userId?: string;
  uploadId?: string;
  name?: string;
  inputName?: string;
  status: string;
  config?: Record<string, unknown>;
  snapshots?: Snapshot[];
  resultUrl?: string;
  converged?: boolean;
  reason?: string;
  finalTime?: number;
  finalStep?: number;
  error?: string;
  createdAt: string;
  updatedAt?: string;
  finishedAt?: string;
  review?: JobReview;
};

export type TrainingClusterJob = {
  clusterId?: string;
  job: Job;
  addedAt: string;
};

export type TrainingRun = {
  id: string;
  clusterId: string;
  status: string;
  metrics: Record<string, unknown>;
  modelArtifact?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string;
};

export type ConfigRecommendation = {
  runId?: string;
  rank: number;
  config: Record<string, unknown>;
  predictedScore: number;
  predictedTags: string[];
  createdAt: string;
};

export type TrainingCluster = {
  id: string;
  userId?: string;
  name: string;
  status: string;
  jobs: TrainingClusterJob[];
  latestRun?: TrainingRun;
  recommendations?: ConfigRecommendation[];
  createdAt: string;
  updatedAt: string;
};

export type JobFrameResponse = {
  label: string;
  url: string;
  text: string;
  isFinal?: boolean;
  simTime?: number;
  step?: number;
};

export type Point3 = {
  x: number;
  y: number;
  z: number;
  fixed: boolean;
  mass: number;
};

export type Edge = {
  a: number;
  b: number;
  restLength: number;
  stiffness: number;
  origin?: "existing" | "generated";
};

export type MeshData = {
  points: Point3[];
  edges: Edge[];
  metadata: Record<string, string>;
};

export type MeshFrame = {
  label: string;
  url: string;
  text: string;
  isFinal?: boolean;
  simTime?: number;
  step?: number;
  pointCloud: MeshData | null;
  loaded: boolean;
  loading: boolean;
  error: AppError | null;
  request: Promise<MeshFrame> | null;
};

export type PointBounds = {
  minX: number;
  minY: number;
  minZ: number;
  maxX: number;
  maxY: number;
  maxZ: number;
};

export type NormalizedPointCloud = {
  points: Point3[];
  min: { x: number; y: number; z: number };
  max: { x: number; y: number; z: number };
  span: number;
};

export type SolverConfig = {
  stiffness: number;
  dampingFactor: number;
  gravity: number;
  airResistanceFactor: number;
  timeStep: number;
  snapshotInterval: number;
  maxSimTime: number;
  maxSteps: number;
  velocityEpsilon: number;
  positionEpsilon: number;
  stableFrames: number;
  springSeed: number;
  maxSpringDist: number;
  maxSpringsPerParticle: number;
  springConnectProb: number;
};

export type AppError = Error & { status?: number; relatedJobIds?: string[] };

export type PreparedMesh = {
  mesh: MeshData;
  text: string;
  sourceName: string;
  uploadId?: string;
  generated: boolean;
};
