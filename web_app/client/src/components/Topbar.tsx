import { observer } from "mobx-react-lite";
import { useStores } from "../stores/store-context";
import "./Topbar.scss";

export const Topbar = observer(function Topbar() {
  const { auth } = useStores();

  return (
    <header className="topbar">
      <div>
        <h1>Mesh3D Solver</h1>
        <p>Pick warehouse meshes, run the server solve, and inspect checkpoint results.</p>
      </div>
      <div className="topbar-actions">
        <span className="status">{auth.serverStatus}</span>
        {auth.currentUser ? <span className="status">{auth.currentUser.username}</span> : null}
        {auth.currentUser ? (
          <button className="secondary" type="button" onClick={() => void auth.logout()}>
            Logout
          </button>
        ) : null}
      </div>
    </header>
  );
});
