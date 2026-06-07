import { createContext, useContext } from "react";
import type { RootStore } from "./root-store";

const StoreContext = createContext<RootStore | null>(null);

export const StoreProvider = StoreContext.Provider;

export function useStores(): RootStore {
  const store = useContext(StoreContext);
  if (!store) {
    throw new Error("StoreProvider is missing.");
  }
  return store;
}
