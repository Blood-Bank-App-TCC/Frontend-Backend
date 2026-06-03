import { create } from "zustand";
import type { AdminUser } from "./types";

const STORAGE_KEY = "bank_darah_admin_session";

interface AuthState {
  token: string | null;
  user: AdminUser | null;
  setSession: (token: string, user: AdminUser) => void;
  clearSession: () => void;
}

function loadSession(): Pick<AuthState, "token" | "user"> {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return { token: null, user: null };
    }
    const session = JSON.parse(raw) as Pick<AuthState, "token" | "user">;
    return { token: session.token ?? null, user: session.user ?? null };
  } catch {
    return { token: null, user: null };
  }
}

const initialSession = loadSession();

export const useAuthStore = create<AuthState>((set) => ({
  token: initialSession.token,
  user: initialSession.user,
  setSession: (token, user) => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ token, user }));
    set({ token, user });
  },
  clearSession: () => {
    window.localStorage.removeItem(STORAGE_KEY);
    set({ token: null, user: null });
  },
}));
