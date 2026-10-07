import { Navigate, Outlet, Route, Routes } from "react-router";
import { OverviewPlaceholder } from "../components/OverviewPlaceholder";
import { RequireRole, RequireSession } from "./session";

// The route table (web LLD section 3). The sign-in and outlets screens arrive
// in W4 and W5; until then their routes hold a heading.
export function AppRoutes() {
  return (
    <Routes>
      <Route path="/sign-in" element={<h1>Sign in</h1>} />
      <Route element={<RequireSession />}>
        <Route element={<Outlet />}>
          <Route index element={<OverviewPlaceholder />} />
          <Route element={<RequireRole role="brand_admin" />}>
            <Route path="/outlets" element={<h1>Outlets</h1>} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
