import { useEffect, useState } from "react";
import { observer } from "mobx-react-lite";
import { AuthPanel } from "./components/AuthPanel";
import { DeleteJobDialog } from "./components/DeleteJobDialog";
import { JobForm } from "./components/JobForm";
import { JobList } from "./components/JobList";
import { LearningPanel } from "./components/LearningPanel";
import { MeshWarehouseDialog } from "./components/MeshWarehouseDialog";
import { Topbar } from "./components/Topbar";
import { ViewerPanel } from "./components/ViewerPanel";
import { useStores } from "./stores/store-context";
import "./App.scss";

export const App = observer(function App() {
  const { auth } = useStores();
  const [activeTab, setActiveTab] = useState<"solver" | "learning">("solver");

  useEffect(() => {
    void auth.init();
  }, [auth]);

  return (
    <>
      <Topbar />
      {auth.isAuthenticated ? (
        <>
          <nav className="workspace-tabs" aria-label="Workspace">
            <button className={activeTab === "solver" ? "active" : "secondary"} type="button" onClick={() => setActiveTab("solver")}>
              Solver
            </button>
            <button className={activeTab === "learning" ? "active" : "secondary"} type="button" onClick={() => setActiveTab("learning")}>
              Learning
            </button>
          </nav>
          {activeTab === "solver" ? (
            <main className="layout">
              <section className="panel controls">
                <JobForm />
                <JobList />
              </section>
              <ViewerPanel />
            </main>
          ) : (
            <LearningPanel />
          )}
        </>
      ) : (
        <AuthPanel />
      )}
      <DeleteJobDialog />
      <MeshWarehouseDialog />
    </>
  );
});
