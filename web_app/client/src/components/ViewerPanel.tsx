import { observer } from "mobx-react-lite";
import { MeshCanvas } from "./MeshCanvas";
import { JobReviewPanel } from "./JobReviewPanel";
import { TimelineControls } from "./TimelineControls";
import { useStores } from "../stores/store-context";
import "./ViewerPanel.scss";
import { ViewerMode } from "../stores/job-store";

export const ViewerPanel = observer(function ViewerPanel() {
  const { jobs, preview, warehouse } = useStores();
  const edgeLegend = getEdgeLegend(jobs.springLegendMesh);

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
      {jobs.reserveSpringDisplay ? (
        <div className={`spring-display-row ${edgeLegend.length === 0 ? "empty-legend" : ""}`}>
          <label className="spring-toggle">
            <input
              disabled={!jobs.canToggleGeneratedSprings}
              type="checkbox"
              checked={preview.includeGeneratedSprings}
              onChange={(event) => preview.setIncludeGeneratedSprings(event.target.checked)}
            />
            {jobs.canToggleGeneratedSprings ? "Enable Springs Generation" : "Spring generation is only available in preview mode"}
          </label>
          {jobs.viewerMode === ViewerMode.Preview ? (
            <div className="spring-legend" aria-label="Spring legend">
              {edgeLegend.map((item) => (
                <span key={item.kind} className={`spring-legend-item ${item.kind}`}>
                  <span className="spring-legend-swatch" />
                  {item.label}
                </span>
              ))}
            </div>
          ) : <div className="spring-legend"></div>}
        </div>
      ) : null}
      <JobReviewPanel />
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
