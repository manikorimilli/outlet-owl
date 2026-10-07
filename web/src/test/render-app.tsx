import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AppRoutes } from "../app/routes";
import { SessionProvider } from "../app/session";
import { LocationProbe } from "./LocationProbe";

// renderApp renders the whole route table at path, with the session provider
// and a probe that shows the current location.
export function renderApp(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SessionProvider>
        <AppRoutes />
        <LocationProbe />
      </SessionProvider>
    </MemoryRouter>,
  );
}
