// Packaged with both complete frontends. Select one before either React router
// starts, regardless of whether Nginx served index.html or control/index.html.
const entries = __KOMARI_ENTRIES__;
const isConsoleRoute = path => /^\/(admin|terminal)(\/|$)/.test(path) ||
  ["/install", "/database-recovery"].includes(path.replace(/\/$/, ""));

if (["/control", "/control/", "/control/index.html"].includes(location.pathname)) {
  const requested = new URLSearchParams(location.search).get("route");
  let target = "/admin/dashboard";
  if (requested?.startsWith("/") && !requested.startsWith("//") && !requested.includes("\\")) {
    const url = new URL(requested, location.origin);
    if (url.origin === location.origin && isConsoleRoute(url.pathname)) {
      target = url.pathname + url.search + url.hash;
    }
  }
  history.replaceState(history.state, "", target);
}

const kind = isConsoleRoute(location.pathname) ? "console" : "lumina";
document.documentElement.dataset.frontend = kind;
document.documentElement.dataset.deployment = "lumina-console";
try {
  const stored = localStorage.getItem("appearance");
  let appearance = stored;
  try { appearance = JSON.parse(stored); } catch { /* Legacy string value. */ }
  const dark = appearance === "dark" || (appearance !== "light" && matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.dataset.appearance = dark ? "dark" : "light";
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
} catch { /* Storage may be unavailable. */ }

async function start() {
  const entry = entries[kind];
  await Promise.all(entry.css.map(href => new Promise((resolve, reject) => {
    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = href;
    link.onload = resolve;
    link.onerror = reject;
    document.head.append(link);
  })));
  await import(entry.js);
}

start().catch(() => {
  const root = document.getElementById("root");
  root.replaceChildren();
  const message = document.createElement("p");
  message.textContent = "页面加载失败，请刷新后重试。";
  const button = document.createElement("button");
  button.textContent = "重新加载";
  button.onclick = () => location.reload();
  root.append(message, button);
});
