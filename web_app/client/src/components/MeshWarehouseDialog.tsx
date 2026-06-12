import { ChangeEvent, MouseEvent, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react-lite";
import { formatDate } from "../lib/format";
import { useStores } from "../stores/store-context";
import type { Upload } from "../types";
import "./MeshWarehouseDialog.scss";

type MeshKind = NonNullable<Upload["meshKind"]>;

export const MeshWarehouseDialog = observer(function MeshWarehouseDialog() {
  const { warehouse } = useStores();
  const fileRef = useRef<HTMLInputElement | null>(null);
  const [selectedMeshKind, setSelectedMeshKind] = useState<MeshKind>("uploaded");
  const deleteUpload = warehouse.deleteOverlayUpload;
  const deleteOverlayOpen = Boolean(deleteUpload);

  useEffect(() => {
    if (!warehouse.pickerOpen) return;
    setSelectedMeshKind(uploadKind(warehouse.selectedUpload));
  }, [warehouse.pickerOpen, warehouse.selectedUpload]);

  useEffect(() => {
    if (!deleteOverlayOpen) return undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        warehouse.closeDeleteOverlay();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [warehouse, deleteOverlayOpen]);

  if (!warehouse.pickerOpen) return null;

  const filteredUploads = warehouse.uploads.filter((upload) => uploadKind(upload) === selectedMeshKind);
  const emptyFilterMessage =
    selectedMeshKind === "generated" ? "No generated meshes saved yet." : "No uploaded meshes saved yet.";

  function uploadNew(event: ChangeEvent<HTMLInputElement>): void {
    const file = event.target.files?.[0] || null;
    warehouse.uploadNew(file);
    event.target.value = "";
  }

  function openDelete(event: MouseEvent<HTMLButtonElement>, upload: Upload): void {
    event.stopPropagation();
    warehouse.openDeleteOverlay(upload);
  }

  return (
    <div className="overlay" role="dialog" aria-modal="true" aria-labelledby="meshWarehouseTitle" onClick={(event) => {
      if (event.target === event.currentTarget) {
        warehouse.closePicker();
      }
    }}>
      <div className="overlay-card mesh-warehouse-overlay">
        <div className="mesh-warehouse-head">
          <div>
            <h2 id="meshWarehouseTitle">Mesh Warehouse</h2>
            <p>Pick a stored mesh or upload a new .msh/.mesh file.</p>
          </div>
          <button className="secondary" type="button" onClick={() => fileRef.current?.click()} disabled={warehouse.uploading}>
            {warehouse.uploading ? "Uploading" : "Upload New"}
          </button>
          <input ref={fileRef} className="hidden" type="file" accept=".msh,.mesh,.txt" onChange={uploadNew} />
        </div>

        {warehouse.error ? <p className="overlay-error">{warehouse.error}</p> : null}

        <fieldset className="mesh-kind-filter" aria-label="Mesh type">
          <label className={`mesh-kind-option ${selectedMeshKind === "uploaded" ? "active" : ""}`}>
            <input
              type="radio"
              name="meshKindFilter"
              value="uploaded"
              checked={selectedMeshKind === "uploaded"}
              onChange={() => setSelectedMeshKind("uploaded")}
            />
            <span>Uploaded</span>
          </label>
          <label className={`mesh-kind-option ${selectedMeshKind === "generated" ? "active" : ""}`}>
            <input
              type="radio"
              name="meshKindFilter"
              value="generated"
              checked={selectedMeshKind === "generated"}
              onChange={() => setSelectedMeshKind("generated")}
            />
            <span>Generated</span>
          </label>
        </fieldset>

        <div className="mesh-warehouse-list">
          {warehouse.loading && warehouse.uploads.length === 0 ? <p className="mesh-warehouse-empty">Loading meshes.</p> : null}
          {!warehouse.loading && warehouse.uploads.length === 0 ? <p className="mesh-warehouse-empty">No meshes saved yet.</p> : null}
          {!warehouse.loading && warehouse.uploads.length > 0 && filteredUploads.length === 0 ? (
            <p className="mesh-warehouse-empty">{emptyFilterMessage}</p>
          ) : null}
          {filteredUploads.map((upload) => (
            <div key={upload.id} className="mesh-warehouse-list-item">
              <div
                key={upload.id}
                className={`mesh-warehouse-card-item ${upload.id === warehouse.selectedUpload?.id ? "active" : ""}`}
                onClick={() => void warehouse.selectUpload(upload)}
              >
                <div className="mesh-warehouse-select" role="button">
                  <span className="mesh-warehouse-title">{upload.fileName}</span>
                  <span className="mesh-warehouse-meta">{uploadMeta(upload)}</span>
                </div>
              </div>
              <button className="danger mesh-warehouse-delete" type="button" onClick={(event) => openDelete(event, upload)}>
                x
              </button>
            </div>
          ))}
        </div>

        <div className="overlay-actions">
          <button className="secondary" type="button" onClick={() => warehouse.closePicker()}>
            Close
          </button>
        </div>
      </div>

      {deleteUpload ? (
        <div className="overlay mesh-delete-overlay" role="dialog" aria-modal="true" aria-labelledby="deleteMeshTitle" onClick={(event) => {
          if (event.target === event.currentTarget) {
            warehouse.closeDeleteOverlay();
          }
        }}>
          <div className="overlay-card">
            <h2 id="deleteMeshTitle">Delete mesh?</h2>
            <p>This removes {deleteUpload.fileName} from your warehouse. Jobs that already use this mesh must be deleted first.</p>
            {warehouse.deleteError ? <p className="overlay-error">{warehouse.deleteError}</p> : null}
            
            <div className="overlay-actions">
              <button className="secondary" type="button" onClick={() => warehouse.closeDeleteOverlay()}>
                Cancel
              </button>
              <button className="danger" type="button" disabled={warehouse.deleting} onClick={() => void warehouse.confirmDelete()}>
                {warehouse.deleting ? "Deleting" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
});

function uploadKind(upload: Upload | null | undefined): MeshKind {
  return upload?.meshKind === "generated" ? "generated" : "uploaded";
}

function uploadMeta(upload: Upload): string {
  const kind = upload.meshKind === "generated" ? "generated (with generated springs)" : "uploaded (raw)";
  const points = `${upload.pointCount || 0} points`;
  const edges = `${upload.edgeCount || 0} springs`;
  return `${kind} - ${points} - ${edges} - ${formatBytes(upload.size)} - ${formatDate(upload.createdAt)}`;
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
