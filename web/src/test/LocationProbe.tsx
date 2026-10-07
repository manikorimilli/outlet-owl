import { useLocation } from "react-router";

// LocationProbe shows the current path and query, so a test can assert where
// a redirect went.
export function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname + location.search}</div>;
}
