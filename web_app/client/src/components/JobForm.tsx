import { FormEvent } from "react";
import { observer } from "mobx-react-lite";
import type { SolverConfig } from "../types";
import { defaultConfig } from "../stores/mesh-preview-store";
import { useStores } from "../stores/store-context";
import "./JobForm.scss";

type NumberField = {
  key: keyof SolverConfig;
  label: string;
  min?: number;
  max?: number;
  step: number;
};

const fields: NumberField[] = [
  { key: "stiffness", label: "Stiffness", min: 0, step: 0.1 },
  { key: "dampingFactor", label: "Damping", min: 0, step: 0.01 },
  { key: "gravity", label: "Gravity", step: 0.1 },
  { key: "airResistanceFactor", label: "Air resistance", min: 0, step: 0.001 },
  { key: "timeStep", label: "Time step", min: 0.0001, step: 0.0001 },
  { key: "snapshotInterval", label: "Snapshot interval", min: 0.05, step: 0.05 },
  { key: "maxSimTime", label: "Max sim time", min: 1, step: 1 },
  { key: "maxSteps", label: "Max steps", min: 1, step: 1 },
  { key: "velocityEpsilon", label: "Velocity epsilon", min: 0, step: 0.0001 },
  { key: "positionEpsilon", label: "Position epsilon", min: 0, step: 0.0001 },
  { key: "stableFrames", label: "Stable frames", min: 1, step: 1 },
  { key: "springSeed", label: "Spring seed", min: 0, step: 1 },
  { key: "maxSpringDist", label: "Max spring dist", min: 0.01, step: 0.01 },
  { key: "maxSpringsPerParticle", label: "Max springs/pt", min: 1, step: 1 },
  { key: "springConnectProb", label: "Connect prob", min: 0, max: 1, step: 0.01 },
];

export const JobForm = observer(function JobForm() {
  const { preview, jobs, warehouse } = useStores();

  async function _submit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    try {
      await jobs.submitJob();
    } catch (error) {
      alert(error instanceof Error ? error.message : "Could not start job.");
    }
  }

  const _isFormValid = Boolean(warehouse.selectedUpload) && preview.preparedMesh?.mesh.edges.length !== 0;

  const _submitBtnTitle = jobs.submitting
    ? "Starting job..."
    : !warehouse.selectedUpload
    ? "Select a mesh to run the job"
    : preview.preparedMesh?.mesh.edges.length === 0
    ? preview.includeGeneratedSprings
      ? "The prepared mesh has no edges. Try adjusting the configuration."
      : "The prepared mesh has no edges. Please enable spring generation to run the job."
    : "";

  return (
    <>
      <h2>New Job</h2>
      <form onSubmit={(event) => _submit(event)}>
        <label>
          Mesh
          <button className="secondary mesh-picker-button" type="button" onClick={() => warehouse.openPicker()}>
            {warehouse.selectedLabel}
          </button>
        </label>

        <label>
          Job name
          <input name="jobName" type="text" value={preview.jobName} onChange={(event) => preview.setJobName(event.target.value)} />
        </label>

        <div className="grid">
          {fields.map((field) => (
            <label key={field.key}>
              {field.label}
              <input
                type="number"
                value={preview.config[field.key]}
                min={field.min ?? undefined}
                max={field.max ?? undefined}
                step={field.step}
                placeholder={String(defaultConfig[field.key])}
                onChange={(event) => preview.setConfigValue(field.key, Number(event.target.value))}
              />
            </label>
          ))}
        </div>

        <p className="spring-status">{preview.status}</p>
        <button className="run-solve-btn" type="submit" 
          disabled={!_isFormValid || jobs.submitting}
          title={_submitBtnTitle}
        >
          {jobs.submitting ? "Starting" : "Run Solve"}
        </button>
        <p className="disclaimer">{_submitBtnTitle}</p>
      </form>
    </>
  );
});
