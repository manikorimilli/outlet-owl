/// <reference types="vite/client" />

interface ImportMetaEnv {
  // "true" on the public demo build only (ADR-0010): prefills the sign-in form.
  readonly VITE_DEMO_PREFILL?: string;
}
