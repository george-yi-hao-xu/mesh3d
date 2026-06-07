import { useEffect, useRef } from "react";
import { observer } from "mobx-react-lite";
import { useStores } from "../stores/store-context";
import "./MeshCanvas.scss";

export const MeshCanvas = observer(function MeshCanvas() {
  const { viewer } = useStores();
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const element = ref.current;
    if (!element) return undefined;
    viewer.mount(element);
    return () => viewer.unmount();
  }, [viewer]);

  return (
    <div ref={ref} className="mesh-canvas">
      <div className={`mesh-message ${viewer.messageHidden ? "hidden" : ""}`}>{viewer.message}</div>
    </div>
  );
});
