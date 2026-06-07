import React from "react";
import { createRoot } from "react-dom/client";
import "./styles/global.scss";
import { App } from "./App";
import { StoreProvider } from "./stores/store-context";
import { RootStore } from "./stores/root-store";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("Missing root element.");
}

createRoot(rootElement).render(
  <React.StrictMode>
    <StoreProvider value={new RootStore()}>
      <App />
    </StoreProvider>
  </React.StrictMode>,
);
