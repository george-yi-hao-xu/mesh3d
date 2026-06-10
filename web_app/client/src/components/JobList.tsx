import { observer } from "mobx-react-lite";
import { formatDate, jobTitle } from "../lib/format";
import { useStores } from "../stores/store-context";
import "./JobList.scss";

export const JobList = observer(function JobList() {
  const { jobs } = useStores();

  return (
    <>
      <div className="job-list-head">
        <h2>Jobs</h2>
        {jobs.hasJobHighlights ? (
          <button className="secondary clear-job-highlights" type="button" onClick={() => jobs.clearJobHighlights()}>
            Clear highlights
          </button>
        ) : null}
      </div>
      <div className="job-list">
        {jobs.jobs.length === 0 ? (
          <p className="job-meta">No jobs yet.</p>
        ) : jobs.jobs.map((job) => (
          <button
            key={job.id}
            type="button"
            className={`job-item ${job.id === jobs.activeJobId ? "active" : ""} ${jobs.isJobHighlighted(job.id) ? "highlighted" : ""}`}
            onClick={() => void jobs.selectJob(job.id)}
          >
            <span className="job-title">{jobTitle(job)}</span>
            <span className="job-meta">
              {job.status} - {job.snapshots?.length || 0} checkpoints - {formatDate(job.createdAt)}
            </span>
          </button>
        ))}
      </div>
    </>
  );
});
