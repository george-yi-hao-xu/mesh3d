import { FormEvent, useState } from "react";
import { observer } from "mobx-react-lite";
import { useStores } from "../stores/store-context";
import "./AuthPanel.scss";

export const AuthPanel = observer(function AuthPanel() {
  const { auth } = useStores();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const isLogin = auth.authMode === "login";

  function submit(event: FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    void auth.submit(username, password);
  }

  return (
    <section className="auth-panel">
      <div className="auth-card">
        <h2>{isLogin ? "Login" : "Create account"}</h2>
        <form onSubmit={submit}>
          <label>
            Username
            <input
              name="username"
              type="text"
              autoComplete="username"
              required
              value={username}
              onChange={(event) => setUsername(event.target.value)}
            />
          </label>
          <label>
            Password
            <input
              name="password"
              type="password"
              autoComplete={isLogin ? "current-password" : "new-password"}
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>
          {auth.error ? <p className="auth-error">{auth.error}</p> : null}
          <button type="submit" disabled={auth.submitting}>
            {auth.submitting ? (isLogin ? "Logging in" : "Registering") : (isLogin ? "Login" : "Register")}
          </button>
        </form>
        <button className="secondary" type="button" onClick={() => auth.toggleAuthMode()}>
          {isLogin ? "Create an account" : "Use existing account"}
        </button>
      </div>
    </section>
  );
});
