import { useEffect } from "react";
import { observer } from "mobx-react-lite";
import { useStores } from "../stores/store-context";
import "./DeleteJobDialog.scss";

export const DeleteJobDialog = observer(function DeleteJobDialog() {
  const { jobs } = useStores();
  const open = Boolean(jobs.deleteOverlayJobId);

  useEffect(() => {
    if (!open) return undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        jobs.closeDeleteOverlay();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [jobs, open]);

  if (!open) return null;

  return (
    <div className="overlay" role="dialog" aria-modal="true" aria-labelledby="deleteOverlayTitle" onClick={(event) => {
      if (event.target === event.currentTarget) {
        jobs.closeDeleteOverlay();
      }
    }}>
      <div className="overlay-card">
        <h2 id="deleteOverlayTitle">Delete job?</h2>
        <p>This removes the job and its saved outputs.</p>
        {jobs.deleteError ? <p className="overlay-error">{jobs.deleteError}</p> : null}
        <div className="overlay-actions">
          <button className="secondary" type="button" onClick={() => jobs.closeDeleteOverlay()}>
            Cancel
          </button>
          <button className="danger" type="button" disabled={jobs.deleting} onClick={() => void jobs.confirmDelete()}>
            Delete
          </button>
        </div>
      </div>
    </div>
  );
});
