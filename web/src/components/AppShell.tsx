import { useState } from "react";
import { Link, NavLink, Outlet, useNavigate } from "react-router";
import { useSession } from "../app/session-context";
import { logout, type Me } from "../features/auth/api";
import { StatusStrip } from "../features/status/StatusStrip";

type NavItem = { to: string; label: string; icon: string };

// Only built routes appear; Digest arrives with phase 6 (web LLD section 3).
// Icon paths come from the mockups. Below 600 px the nav is a bottom tab bar.
const overview: NavItem = { to: "/", label: "Overview", icon: "M4 20V10M10 20V4M16 20v-7M22 20H2" };
const outlets: NavItem = {
  to: "/outlets",
  label: "Outlets",
  icon: "M3 9l2-5h14l2 5M4 9v11h16V9M9 20v-6h6v6",
};

const importNav: NavItem = {
  to: "/import",
  label: "Import",
  icon: "M12 15V3M7 8l5-5 5 5M4 15v5h16v-5",
};

const reviewsNav: NavItem = {
  to: "/reviews",
  label: "Reviews",
  icon: "M4 5h16M4 10h16M4 15h10M4 20h7",
};
const themesNav: NavItem = {
  to: "/themes",
  label: "Themes",
  icon: "M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z",
};

function navFor(role: Me["role"]): NavItem[] {
  const shared = [overview, reviewsNav, themesNav];
  return role === "brand_admin" ? [...shared, outlets, importNav] : shared;
}

function whoLabel(me: Me): string {
  if (me.role === "brand_admin") {
    return `${me.name}, brand admin`;
  }
  return me.outlet ? `${me.name}, manager, ${me.outlet.name}` : `${me.name}, manager`;
}

// AppShell is the frame of every signed-in screen: the navy rail with the
// side nav, the top bar with the brand, the person and Sign out.
export function AppShell() {
  const { session, signedOut } = useSession();
  const navigate = useNavigate();
  const [signingOut, setSigningOut] = useState(false);

  if (session.status !== "signed-in") {
    return null; // RequireSession renders the shell only for a signed-in user
  }
  const { me } = session;

  async function signOut() {
    setSigningOut(true);
    // Signed out either way: if the call fails the cookie expires on its own.
    await logout().catch(() => undefined);
    signedOut();
    navigate("/sign-in", { replace: true });
  }

  return (
    <div className="app">
      <a className="skip" href="#main">
        Skip to content
      </a>
      <div className="rail">
        <Link className="brand" to="/">
          <svg className="logo" viewBox="0 0 32 32" aria-hidden="true">
            <rect className="logo-b" x="2" y="3" width="28" height="22" rx="9" />
            <path className="logo-b" d="M9 23l-1.5 7 8-7z" />
            <path
              className="logo-s"
              d="M16 8.5l2 4.1 4.5.6-3.3 3.1.8 4.4-4-2.1-4 2.1.8-4.4-3.3-3.1 4.5-.6z"
            />
          </svg>
          <span>OutletOwl</span>
        </Link>
        <nav className="sidenav" aria-label="Sections">
          {navFor(me.role).map((item) => (
            <NavLink key={item.to} to={item.to} end>
              <svg
                viewBox="0 0 24 24"
                aria-hidden="true"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.75"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d={item.icon} />
              </svg>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </div>
      <header className="topbar">
        <span className="brand-sub">{me.brand.name}</span>
        <StatusStrip />
        <span className="spacer" />
        <span className="muted who">{whoLabel(me)}</span>
        <button
          type="button"
          className="btn btn-secondary btn-small"
          onClick={signOut}
          disabled={signingOut}
        >
          Sign out
        </button>
      </header>
      <main id="main" className="main">
        <Outlet />
      </main>
    </div>
  );
}
