// The seeded brand admin, filled into the sign-in form for a demo.
export const demoAdmin = { email: "ritika.rao@example.in", password: "outletowl-demo" };

export type DemoSignIn = {
  // account fills the form; null leaves it empty.
  account: typeof demoAdmin | null;
  // note shows "this is a demo" on the page, only on the public demo.
  note: boolean;
};

// noDemo is the empty form, for builds without the demo.
export const noDemo: DemoSignIn = { account: null, note: false };

// demoSignIn decides whether the sign-in form is prefilled. Under the Vite dev
// server (make web-dev) it is, quietly. A build with VITE_DEMO_PREFILL=true
// (set only on the Vercel demo, ADR-0010) prefills and says it is a demo.
// Tests run in "test" mode and a normal build in "production", so neither
// carries the account unless the flag is set.
export function demoSignIn(env: { MODE: string; VITE_DEMO_PREFILL?: string }): DemoSignIn {
  if (env.VITE_DEMO_PREFILL === "true") {
    return { account: demoAdmin, note: true };
  }
  if (env.MODE === "development") {
    return { account: demoAdmin, note: false };
  }
  return noDemo;
}
