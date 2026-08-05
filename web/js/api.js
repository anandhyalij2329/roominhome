const API = "";

function getToken() {
  return localStorage.getItem("adnivara_token") || localStorage.getItem("roominhome_token") || "";
}

function getUser() {
  try {
    return JSON.parse(
      localStorage.getItem("adnivara_user") || localStorage.getItem("roominhome_user") || "null"
    );
  } catch {
    return null;
  }
}

function setSession(token, user) {
  localStorage.setItem("adnivara_token", token);
  localStorage.setItem("adnivara_user", JSON.stringify(user));
  localStorage.removeItem("roominhome_token");
  localStorage.removeItem("roominhome_user");
}

function clearSession() {
  localStorage.removeItem("adnivara_token");
  localStorage.removeItem("adnivara_user");
  localStorage.removeItem("roominhome_token");
  localStorage.removeItem("roominhome_user");
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = headers["Content-Type"] || "application/json";
  }
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(API + path, { ...options, headers });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { error: text || "invalid response" };
  }
  if (!res.ok) {
    const err = new Error((data && data.error) || res.statusText || "request failed");
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

function formatRent(n) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0,
  }).format(n || 0);
}

function coverUrl(property) {
  const media = property.media || [];
  const cover = media.find((m) => m.is_cover) || media.find((m) => m.type === "photo");
  if (cover) return cover.url;
  return "https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&w=800&q=70";
}

function locationLine(p) {
  return [p.locality, p.city].filter(Boolean).join(", ");
}

function qs(name) {
  return new URLSearchParams(location.search).get(name);
}

function typeLabel(t) {
  return ({ room: "Room", home: "Home", pg: "PG", shop: "Shop" }[t] || t || "").toUpperCase();
}

function listingCardHTML(p, i) {
  const dist = p.distance_km ? ` · ${p.distance_km} km` : "";
  return `
    <a class="listing" href="/property.html?id=${p.id}" style="animation-delay:${(i % 8) * 0.05}s">
      <div class="listing-media"><img src="${coverUrl(p)}" alt="" loading="lazy" /></div>
      <div class="listing-body">
        <div class="listing-type">${typeLabel(p.type)}</div>
        <h3>${p.title}</h3>
        <div class="meta">${locationLine(p)}${dist}</div>
        <div class="rent">${formatRent(p.rent)} / month</div>
      </div>
    </a>`;
}

/**
 * Header: Room | Home | PG | Shop  (+ account links)
 */
function renderNav(active) {
  const user = getUser();
  const links = document.getElementById("nav-links");
  if (!links) return;

  const categories = [
    { href: "/listings.html?type=room", label: "Room", key: "room" },
    { href: "/listings.html?type=home", label: "Home", key: "home" },
    { href: "/listings.html?type=pg", label: "PG", key: "pg" },
    { href: "/listings.html?type=shop", label: "Shop", key: "shop" },
  ];

  let html = `<div class="nav-cats">`;
  html += categories
    .map((i) => {
      const on =
        active === i.key ||
        (active === "listings" && qs("type") === i.key) ||
        (typeof active === "string" && active.startsWith("type:") && active.slice(5) === i.key);
      return `<a class="nav-cat${on ? " is-active" : ""}" href="${i.href}">${i.label}</a>`;
    })
    .join("");
  html += `</div><div class="nav-account">`;

  if (user?.role === "owner") {
    html += `<a href="/owner.html" class="${active === "owner" ? "is-active" : ""}">My listings</a>`;
  }
  if (user?.role === "seeker") {
    html += `<a href="/bookings.html" class="${active === "bookings" ? "is-active" : ""}">Bookings</a>`;
  }
  if (user?.role === "admin") {
    html += `<a href="/admin.html" class="${active === "admin" ? "is-active" : ""}">Admin</a>`;
  }

  if (user) {
    html += `<span class="muted nav-user">${user.name}</span>`;
    html += `<button type="button" class="btn btn-ghost" id="logout-btn">Log out</button>`;
  } else {
    html += `<a href="/login.html">Log in</a>`;
    html += `<a class="btn btn-primary" href="/register.html">Join</a>`;
  }
  html += `</div>`;

  links.innerHTML = html;

  const logout = document.getElementById("logout-btn");
  if (logout) {
    logout.addEventListener("click", () => {
      clearSession();
      location.href = "/";
    });
  }
}
