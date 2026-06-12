import { FormEvent, useEffect, useState } from "react";
import { observer } from "mobx-react-lite";
import { formatDate, jobTitle } from "../lib/format";
import { useStores } from "../stores/store-context";
import type { ConfigRecommendation } from "../types";
import "./LearningPanel.scss";

export const LearningPanel = observer(function LearningPanel() {
  const { learning, warehouse } = useStores();
  const cluster = learning.activeCluster;
  const [newName, setNewName] = useState("Config training");
  const [rename, setRename] = useState("");

  useEffect(() => {
    void learning.refreshClusters();
  }, [learning]);

  useEffect(() => {
    setRename(cluster?.name || "");
  }, [cluster?.id, cluster?.name]);

  async function create(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await learning.createCluster(newName);
  }

  async function saveRename(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    await learning.renameActive(rename);
  }

  const reviewedJobs = learning.reviewedJobs;
  const selectedIds = learning.activeClusterJobIds;

  return (
    <main className="learning-layout">
      <section className="panel learning-sidebar">
        <div className="learning-head">
          <div>
            <h2>Learning Clusters</h2>
            <p>Pack reviewed jobs, train a tabular model, and rank solver configs.</p>
          </div>
          <button className="secondary" type="button" onClick={() => void learning.refreshClusters()} disabled={learning.loading}>
            Refresh
          </button>
        </div>

        <form className="cluster-create" onSubmit={(event) => void create(event)}>
          <label>
            New cluster
            <input value={newName} onChange={(event) => setNewName(event.target.value)} />
          </label>
          <button type="submit" disabled={learning.saving}>Create</button>
        </form>

        <div className="cluster-list">
          {learning.clusters.length === 0 ? <p>No clusters yet.</p> : null}
          {learning.clusters.map((item) => (
            <button
              key={item.id}
              className={`cluster-item ${item.id === cluster?.id ? "active" : ""}`}
              type="button"
              onClick={() => learning.selectCluster(item.id)}
            >
              <span className="cluster-title">{item.name}</span>
              <span>{item.status} - {item.jobs.length} jobs</span>
            </button>
          ))}
        </div>
      </section>

      <section className="panel learning-main">
        {!cluster ? (
          <p>Create a cluster to begin.</p>
        ) : (
          <>
            <div className="learning-head">
              <div>
                <h2>{cluster.name}</h2>
                <p>{cluster.status} - updated {formatDate(cluster.updatedAt)}</p>
              </div>
              <button className="danger" type="button" onClick={() => void learning.deleteActive()} disabled={learning.saving}>
                Delete
              </button>
            </div>

            <form className="cluster-rename" onSubmit={(event) => void saveRename(event)}>
              <label>
                Cluster name
                <input value={rename} onChange={(event) => setRename(event.target.value)} />
              </label>
              <button className="secondary" type="submit" disabled={learning.saving || !rename.trim()}>
                Save
              </button>
            </form>

            <div className="learning-actions">
              <button type="button" onClick={() => void learning.trainActive()} disabled={learning.training || cluster.jobs.length < 20}>
                {learning.training ? "Training" : "Train Model"}
              </button>
              <button
                className="secondary"
                type="button"
                onClick={() => void learning.recommendForSelectedUpload()}
                disabled={learning.recommending || cluster.latestRun?.status !== "trained"}
              >
                {learning.recommending ? "Ranking" : "Recommend Configs"}
              </button>
              <span>{warehouse.selectedUpload ? `Target mesh: ${warehouse.selectedUpload.fileName}` : "Target mesh: none selected"}</span>
            </div>

            {cluster.latestRun ? <RunSummary run={cluster.latestRun} /> : null}
            {learning.error ? <p className="learning-error">{learning.error}</p> : null}

            <section className="learning-section">
              <div className="learning-head">
                <div>
                  <h3>Reviewed Jobs</h3>
                  <p>{selectedIds.size} selected. Training requires at least 20 reviewed jobs.</p>
                </div>
              </div>
              <div className="reviewed-job-list">
                {reviewedJobs.length === 0 ? <p>No reviewed jobs yet.</p> : null}
                {reviewedJobs.map((job) => (
                  <button
                    key={job.id}
                    className={`reviewed-job ${selectedIds.has(job.id) ? "selected" : ""}`}
                    type="button"
                    onClick={() => void learning.toggleJob(job.id)}
                    disabled={learning.saving}
                  >
                    <span>{jobTitle(job)}</span>
                    <span>score {job.review?.score} - {(job.review?.tags || []).join(", ") || "no tags"}</span>
                  </button>
                ))}
              </div>
            </section>

            <section className="learning-section">
              <h3>Recommended Configs</h3>
              <div className="recommendation-list">
                {(cluster.recommendations || []).length === 0 ? <p>No recommendations yet.</p> : null}
                {(cluster.recommendations || []).map((rec) => (
                  <RecommendationCard key={`${rec.runId}-${rec.rank}`} rec={rec} onApply={() => learning.applyConfig(rec.config)} />
                ))}
              </div>
            </section>
          </>
        )}
      </section>
    </main>
  );
});

function RunSummary({ run }: { run: { status: string; error?: string; metrics?: Record<string, unknown> } }) {
  return (
    <div className={`run-summary ${run.status}`}>
      <strong>Latest run: {run.status}</strong>
      {run.error ? <span>{run.error}</span> : null}
      {run.metrics ? <span>{formatMetrics(run.metrics)}</span> : null}
    </div>
  );
}

function RecommendationCard({ rec, onApply }: { rec: ConfigRecommendation; onApply: () => void }) {
  const entries = Object.entries(rec.config).slice(0, 8);
  return (
    <article className="recommendation-card">
      <div>
        <h4>Rank {rec.rank} - predicted {rec.predictedScore.toFixed(2)}</h4>
        <p>{rec.predictedTags.length ? rec.predictedTags.join(", ") : "No likely tags"}</p>
      </div>
      <dl>
        {entries.map(([key, value]) => (
          <div key={key}>
            <dt>{key}</dt>
            <dd>{String(value)}</dd>
          </div>
        ))}
      </dl>
      <button type="button" onClick={onApply}>Apply to Solver</button>
    </article>
  );
}

function formatMetrics(metrics: Record<string, unknown>): string {
  return Object.entries(metrics)
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(",") : String(value)}`)
    .join(" - ");
}
