import { useEffect } from "react";
import { observer } from "mobx-react-lite";
import { AuthPanel } from "./components/AuthPanel";
import { DeleteJobDialog } from "./components/DeleteJobDialog";
import { JobForm } from "./components/JobForm";
import { JobList } from "./components/JobList";
import { Topbar } from "./components/Topbar";
import { ViewerPanel } from "./components/ViewerPanel";
import { useStores } from "./stores/store-context";
import "./App.scss";

export const App = observer(function App() {
  const { auth } = useStores();

  useEffect(() => {
    void auth.init();
  }, [auth]);

  return (
    <>
      <Topbar />
      {auth.isAuthenticated ? (
        <main className="layout">
          <section className="panel controls">
            <JobForm />
            <JobList />
          </section>
          <ViewerPanel />
        </main>
      ) : (
        <AuthPanel />
      )}
      <DeleteJobDialog />
    </>
  );
});
