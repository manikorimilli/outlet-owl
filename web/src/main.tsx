import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
// Fonts are bundled, never loaded from a CDN (HLD section 3, phase 4 LLD).
import "@fontsource/plus-jakarta-sans/400.css";
import "@fontsource/plus-jakarta-sans/500.css";
import "@fontsource/plus-jakarta-sans/600.css";
import "@fontsource/plus-jakarta-sans/700.css";
import "@fontsource/plus-jakarta-sans/800.css";
import "@fontsource/noto-sans-devanagari/400.css";
import "@fontsource/noto-sans-devanagari/600.css";
import "@fontsource/noto-sans-devanagari/700.css";
import "./styles/tokens.css";
import "./styles/app.css";

const root = document.getElementById("root");
if (!root) {
  throw new Error("index.html has no #root element");
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
