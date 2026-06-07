import { observer } from "mobx-react-lite";
import { MeshCanvas } from "./MeshCanvas";
import { TimelineControls } from "./TimelineControls";
import { useStores } from "../stores/store-context";
import "./ViewerPanel.scss";

export const ViewerPanel = observer(function ViewerPanel() {
  const { jobs, preview, warehouse } = useStores();
  const edgeLegend = getEdgeLegend(jobs.activePreview?.mesh || jobs.selectedFrame?.pointCloud || null);
  const canToggleGeneratedSprings = Boolean(preview.sourceMesh && (jobs.activePreview || !jobs.activeJob));

  return (
    <section className="panel viewer">
      <div className="viewer-head">
        <div>
          <h2>{jobs.activeTitle}</h2>
          <p>{jobs.activeMeta}</p>
        </div>
        <div className="viewer-actions">
          {warehouse.canSaveGeneratedMesh ? (
            <button className="secondary" type="button" disabled={warehouse.savingGenerated} onClick={() => void warehouse.saveCurrentPreview()}>
              {warehouse.savingGenerated ? "Saving" : "Save to Warehouse"}
            </button>
          ) : null}
          {jobs.downloadUrl ? (
            <a className="download" href={jobs.downloadUrl} download={jobs.downloadName}>
              Download
            </a>
          ) : null}
          {jobs.canDeleteActiveJob ? (
            <button className="danger" type="button" onClick={() => jobs.openDeleteOverlay()}>
              Delete
            </button>
          ) : null}
        </div>
      </div>

      {jobs.activeInputName ? <p className="input-name">{jobs.activeInputName}</p> : null}
      {edgeLegend.length > 0 || canToggleGeneratedSprings ? (
        <div className="spring-display-row">
          {canToggleGeneratedSprings ? (
            <label className="spring-toggle">
              <input
                type="checkbox"
                checked={preview.includeGeneratedSprings}
                onChange={(event) => preview.setIncludeGeneratedSprings(event.target.checked)}
              />
              Enable Springs Generation
            </label>
          ) : null}
          {edgeLegend.length > 0 ? (
            <div className="spring-legend" aria-label="Spring legend">
              {edgeLegend.map((item) => (
                <span key={item.kind} className={`spring-legend-item ${item.kind}`}>
                  <span className="spring-legend-swatch" />
                  {item.label}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      <TimelineControls />
      <MeshCanvas />
      <details className="raw-mesh">
        <summary>Raw mesh artifact</summary>
        <pre className="mesh-preview">{jobs.rawPreviewText}</pre>
      </details>
    </section>
  );
});

function getEdgeLegend(mesh: { edges: Array<{ origin?: "existing" | "generated" }> } | null): Array<{ kind: "existing" | "generated"; label: string }> {
  if (!mesh?.edges.length) return [];
  const hasExisting = mesh.edges.some((edge) => edge.origin !== "generated");
  const hasGenerated = mesh.edges.some((edge) => edge.origin === "generated");
  const legend: Array<{ kind: "existing" | "generated"; label: string }> = [];
  if (hasExisting) legend.push({ kind: "existing", label: "Existing" });
  if (hasGenerated) legend.push({ kind: "generated", label: "Generated" });
  return legend;
}
