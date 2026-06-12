import { FormEvent, useEffect, useState } from "react";
import { observer } from "mobx-react-lite";
import type { Job } from "../types";
import { useStores } from "../stores/store-context";
import "./JobReviewPanel.scss";

const reviewTags = [
  "stable",
  "good-result",
  "too-slow",
  "too-stretchy",
  "collapsed",
  "bad-springs",
  "nice-motion",
];

export const JobReviewPanel = observer(function JobReviewPanel() {
  const { jobs } = useStores();
  const job = jobs.activeJob;
  const [score, setScore] = useState(0);
  const [tags, setTags] = useState<string[]>([]);
  const [note, setNote] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    setScore(job?.review?.score || 0);
    setTags(job?.review?.tags || []);
    setNote(job?.review?.note || "");
    setMessage("");
  }, [job?.id, job?.review?.score, job?.review?.note, job?.review?.tags]);

  if (!canReview(job)) return null;

  async function submit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (!score) {
      setMessage("Choose a score before saving.");
      return;
    }
    try {
      await jobs.saveReview(score, tags, note);
      setMessage("Review saved.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not save review.");
    }
  }

  function toggleTag(tag: string): void {
    setTags((current) => current.includes(tag)
      ? current.filter((item) => item !== tag)
      : [...current, tag]);
  }

  return (
    <section className="job-review" aria-label="Job review">
      <div className="job-review-head">
        <h3>Review</h3>
        <p>Score this result to build training labels.</p>
      </div>
      <form className="job-review-form" onSubmit={(event) => void submit(event)}>
        <div className="score-row" role="radiogroup" aria-label="Review score">
          {[1, 2, 3, 4, 5].map((value) => (
            <button
              key={value}
              type="button"
              className={`score-button ${score === value ? "selected" : ""}`}
              aria-pressed={score === value}
              onClick={() => setScore(value)}
            >
              {value}
            </button>
          ))}
        </div>
        <div className="tag-row" aria-label="Review tags">
          {reviewTags.map((tag) => (
            <button
              key={tag}
              type="button"
              className={`tag-button ${tags.includes(tag) ? "selected" : ""}`}
              aria-pressed={tags.includes(tag)}
              onClick={() => toggleTag(tag)}
            >
              {tag}
            </button>
          ))}
        </div>
        <label>
          Note
          <textarea value={note} maxLength={500} onChange={(event) => setNote(event.target.value)} />
        </label>
        <div className="review-actions">
          <button type="submit" disabled={jobs.savingReview || !score}>
            {jobs.savingReview ? "Saving" : "Save Review"}
          </button>
          {message ? <span className="review-message">{message}</span> : null}
        </div>
      </form>
    </section>
  );
});

function canReview(job: Job | null): boolean {
  return Boolean(job && (job.status === "done" || job.status === "failed"));
}
