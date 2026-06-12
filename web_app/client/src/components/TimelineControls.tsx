import { observer } from "mobx-react-lite";
import { FaBackward, FaBackwardStep, FaForward, FaForwardStep, FaPause, FaPlay } from "react-icons/fa6";
import { useStores } from "../stores/store-context";
import "./TimelineControls.scss";

export const TimelineControls = observer(function TimelineControls() {
  const { jobs } = useStores();
  const frames = jobs.activeFrames;
  const selectedIndex = jobs.selectedFrameIndex;
  const selectedFrame = jobs.selectedFrame;
  const disabled = frames.length < 2;

  if (frames.length === 0) {
    return <div className="tabs" />;
  }

  return (
    <div className="tabs frame-control">
      <div className="frame-head">
        <div className="frame-tools">
          <div className="transport-controls" aria-label="Timeline controls">
            <button className="transport-button" type="button" disabled={disabled} onClick={() => jobs.selectFrameAt_safe(0)} aria-label="First frame" title="First frame">
              <FaBackwardStep aria-hidden="true" />
            </button>
            <button className="transport-button" type="button" disabled={disabled} onClick={() => jobs.selectFrameAt_safe(selectedIndex - 1)} aria-label="Previous frame" title="Previous frame">
              <FaBackward aria-hidden="true" />
            </button>
            <button className="transport-button" type="button" disabled={disabled} onClick={() => jobs.togglePlayback()} aria-label={jobs.playback ? "Pause playback" : "Play frames"} title={jobs.playback ? "Pause playback" : "Play frames"}>
              {jobs.playback ? <FaPause aria-hidden="true" /> : <FaPlay aria-hidden="true" />}
            </button>
            {/* <button className="transport-button" type="button" disabled={disabled} onClick={() => {
              jobs.stopPlayback();
              jobs.selectFrameAt(0);
            }}>
              Stop
            </button> */}
            <button className="transport-button" type="button" disabled={disabled} onClick={() => jobs.selectFrameAt_safe(selectedIndex + 1)} aria-label="Next frame" title="Next frame">
              <FaForward aria-hidden="true" />
            </button>
            <button className="transport-button" type="button" disabled={disabled} onClick={() => jobs.selectFrameAt_safe(frames.length - 1)} aria-label="Last frame" title="Last frame">
              <FaForwardStep aria-hidden="true" />
            </button>
          </div>
          <span className="frame-title">Time frame</span>
        </div>
        <span className="frame-label">{selectedFrame ? `${selectedFrame.label} (${selectedIndex + 1}/${frames.length})` : ""}</span>
      </div>
      <input
        className="frame-slider"
        type="range"
        min="0"
        max={frames.length - 1}
        step="1"
        value={selectedIndex}
        onChange={(event) => {
          jobs.stopPlayback();
          jobs.selectFrameAt_safe(Number(event.target.value));
        }}
      />
      <div className="frame-scale">
        <span>{frames[0].label}</span>
        <span>{frames[frames.length - 1].label}</span>
      </div>
    </div>
  );
});
