import { Navigate, Route, Routes } from "react-router";
import { AppShell } from "../components/AppShell";
import { OverviewPlaceholder } from "../components/OverviewPlaceholder";
import { SignInPage } from "../features/auth/SignInPage";
import { ImportPage } from "../features/imports/ImportPage";
import { OutletsPage } from "../features/outlets/OutletsPage";
import { RequireRole, RequireSession } from "./session";

// The route table (web LLD section 3).
export function AppRoutes() {
  return (
    <Routes>
      <Route path="/sign-in" element={<SignInPage />} />
      <Route element={<RequireSession />}>
        <Route element={<AppShell />}>
          <Route index element={<OverviewPlaceholder />} />
          <Route element={<RequireRole role="brand_admin" />}>
            <Route path="/outlets" element={<OutletsPage />} />
            <Route path="/import" element={<ImportPage />} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
