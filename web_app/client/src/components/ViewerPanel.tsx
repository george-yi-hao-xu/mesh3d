import { observer } from "mobx-react-lite";
import { MeshCanvas } from "./MeshCanvas";
import { TimelineControls } from "./TimelineControls";
import { useStores } from "../stores/store-context";
import "./ViewerPanel.scss";

export const ViewerPanel = observer(function ViewerPanel() {
  const { jobs } = useStores();

  return (
    <section className="panel viewer">
      <div className="viewer-head">
        <div>
          <h2>{jobs.activeTitle}</h2>
          <p>{jobs.activeMeta}</p>
        </div>
        <div className="viewer-actions">
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
      <TimelineControls />
      <MeshCanvas />
      <details className="raw-mesh">
        <summary>Raw mesh artifact</summary>
        <pre className="mesh-preview">{jobs.rawPreviewText}</pre>
      </details>
    </section>
  );
});
