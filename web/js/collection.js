const META = {
  room: {
    title: "Room collection",
    desc: "Private and shared rooms with parking, amenities and clear rent terms.",
    img: "https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1800&q=80",
    subs: [
      { label: "All rooms", q: {} },
      { label: "Private", q: { sharing_type: "private" } },
      { label: "1 sharing", q: { sharing_type: "1_sharing" } },
      { label: "2 sharing", q: { sharing_type: "2_sharing" } },
      { label: "Bachelor", q: { preferred_tenant: "bachelor" } },
      { label: "Family", q: { preferred_tenant: "family" } },
    ],
  },
  home: {
    title: "Home collection",
    desc: "Flats and houses — BHK options for families and long-term stays.",
    img: "https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?auto=format&fit=crop&w=1800&q=80",
    subs: [
      { label: "All homes", q: {} },
      { label: "Family", q: { preferred_tenant: "family" } },
      { label: "Furnished", q: {} },
      { label: "2-wheeler parking", q: { parking_two_wheeler: "true" } },
      { label: "4-wheeler parking", q: { parking_four_wheeler: "true" } },
    ],
  },
  pg: {
    title: "PG / Co-living collection",
    desc: "1 sharing, 2 sharing, 3 sharing — with meals, gender and parking filters.",
    img: "https://images.unsplash.com/photo-1555854877-bab0e564b8d5?auto=format&fit=crop&w=1800&q=80",
    subs: [
      { label: "All PG", q: {} },
      { label: "1 sharing", q: { sharing_type: "1_sharing" } },
      { label: "2 sharing", q: { sharing_type: "2_sharing" } },
      { label: "3 sharing", q: { sharing_type: "3_sharing" } },
      { label: "Private", q: { sharing_type: "private" } },
      { label: "Student", q: { preferred_tenant: "student" } },
      { label: "Bachelor", q: { preferred_tenant: "bachelor" } },
    ],
  },
  shop: {
    title: "Shop collection",
    desc: "Retail, office, showroom and commercial spaces with frontage and area.",
    img: "https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1800&q=80",
    subs: [
      { label: "All shops", q: {} },
      { label: "Retail", q: { shop_category: "retail" } },
      { label: "Office", q: { shop_category: "office" } },
      { label: "Showroom", q: { shop_category: "showroom" } },
      { label: "Warehouse", q: { shop_category: "warehouse" } },
      { label: "Clinic", q: { shop_category: "clinic" } },
    ],
  },
};

const type = (qs("type") || "room").toLowerCase();
const meta = META[type] || META.room;
let extraQuery = {};

document.title = `${meta.title} - Adnivara`;
renderNav(type);
document.getElementById("hero-img").src = meta.img;
document.getElementById("hero-title").textContent = meta.title;
document.getElementById("hero-desc").textContent = meta.desc;

document.querySelectorAll("[data-chip]").forEach((a) => {
  if (a.dataset.chip === type) a.classList.add("is-active");
});

const subEl = document.getElementById("sub-options");
subEl.innerHTML = meta.subs
  .map(
    (s, i) =>
      `<button type="button" class="${i === 0 ? "is-active" : ""}" data-i="${i}">${s.label}</button>`
  )
  .join("");

subEl.addEventListener("click", (e) => {
  const btn = e.target.closest("button[data-i]");
  if (!btn) return;
  subEl.querySelectorAll("button").forEach((b) => b.classList.remove("is-active"));
  btn.classList.add("is-active");
  extraQuery = { ...(meta.subs[Number(btn.dataset.i)].q || {}) };
  load();
});

const form = document.getElementById("filters");
form.addEventListener("submit", (e) => {
  e.preventDefault();
  load();
});

function listingCard(p, i) {
  return `
    <a class="listing" href="/property.html?id=${p.id}" style="animation-delay:${(i % 8) * 0.05}s">
      <div class="listing-media"><img src="${coverUrl(p)}" alt="" loading="lazy" /></div>
      <div class="listing-body">
        <div class="listing-type">${typeLabel(p.type)}</div>
        <h3>${p.title}</h3>
        <div class="meta">${locationLine(p)}${p.sharing_type ? " - " + p.sharing_type.replaceAll("_", " ") : ""}</div>
        <div class="rent">${formatRent(p.rent)} / month</div>
      </div>
    </a>`;
}

async function load() {
  const results = document.getElementById("results");
  results.innerHTML = `<div class="empty">Loading…</div>`;

  const fd = new FormData(form);
  const q = new URLSearchParams();
  q.set("type", type);
  q.set("status", "available");
  for (const [k, v] of fd.entries()) {
    if (String(v).trim()) q.set(k, String(v).trim());
  }
  Object.entries(extraQuery).forEach(([k, v]) => {
    if (v) q.set(k, v);
  });

  try {
    const list = await api(`/api/properties?${q}`);
    document.getElementById("hero-count").textContent =
      `${list.length} listing${list.length === 1 ? "" : "s"} in this collection`;
    if (!list.length) {
      results.innerHTML = `<div class="empty">No listings in this collection yet. <a href="/register.html?role=owner">List one</a>.</div>`;
      return;
    }
    results.innerHTML = list.map(listingCard).join("");
  } catch (e) {
    results.innerHTML = `<div class="empty">${e.message}</div>`;
    document.getElementById("hero-count").textContent = "Could not load";
  }
}

load();
