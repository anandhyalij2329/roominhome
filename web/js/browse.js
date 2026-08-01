renderNav("browse");

const form = document.getElementById("filters");
const results = document.getElementById("results");
let map, markersLayer;
let center = { lat: 18.5204, lng: 73.8567 };

const params = new URLSearchParams(location.search);
["type", "city", "locality", "preferred_tenant", "sharing_type"].forEach((k) => {
  if (params.get(k) && form.elements[k]) form.elements[k].value = params.get(k);
});

function initMap() {
  map = L.map("browse-map").setView([center.lat, center.lng], 12);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    attribution: "&copy; OpenStreetMap",
  }).addTo(map);
  markersLayer = L.layerGroup().addTo(map);
  map.on("click", (e) => {
    center = { lat: e.latlng.lat, lng: e.latlng.lng };
    loadNearby();
  });
}

function listingCard(p, i) {
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

function plotMarkers(list) {
  markersLayer.clearLayers();
  list.forEach((p) => {
    if (!p.latitude && !p.longitude) return;
    const m = L.marker([p.latitude, p.longitude]).addTo(markersLayer);
    m.bindPopup(`<strong>${p.title}</strong><br>${formatRent(p.rent)}<br><a href="/property.html?id=${p.id}">View</a>`);
  });
}

async function loadFiltered() {
  const fd = new FormData(form);
  const q = new URLSearchParams();
  for (const [k, v] of fd.entries()) {
    if (String(v).trim()) q.set(k, String(v).trim());
  }
  q.set("status", "available");
  results.innerHTML = `<div class="empty">Searching…</div>`;
  try {
    const list = await api(`/api/properties?${q}`);
    if (!list.length) {
      results.innerHTML = `<div class="empty">No properties match these filters.</div>`;
      plotMarkers([]);
      return;
    }
    results.innerHTML = list.map(listingCard).join("");
    plotMarkers(list);
    if (list[0]?.latitude) {
      map.setView([list[0].latitude, list[0].longitude], 12);
    }
  } catch (e) {
    results.innerHTML = `<div class="empty">${e.message}</div>`;
  }
}

async function loadNearby() {
  const radius = document.getElementById("radius").value || 5;
  results.innerHTML = `<div class="empty">Finding nearby…</div>`;
  try {
    const type = form.elements.type.value;
    let path = `/api/map/nearby?lat=${center.lat}&lng=${center.lng}&radius_km=${radius}`;
    if (type) path += `&type=${encodeURIComponent(type)}`;
    const data = await api(path);
    const list = data.properties || [];
    markersLayer.clearLayers();
    L.circle([center.lat, center.lng], { radius: radius * 1000, color: "#1f6b57", fillOpacity: 0.08 }).addTo(markersLayer);
    L.circleMarker([center.lat, center.lng], { radius: 8, color: "#1f6b57" }).addTo(markersLayer);
    if (!list.length) {
      results.innerHTML = `<div class="empty">Nothing within ${radius} km. Click the map to move the search.</div>`;
      return;
    }
    results.innerHTML = list.map(listingCard).join("");
    list.forEach((p) => {
      if (!p.latitude && !p.longitude) return;
      const m = L.marker([p.latitude, p.longitude]).addTo(markersLayer);
      m.bindPopup(`<strong>${p.title}</strong><br>${formatRent(p.rent)}<br><a href="/property.html?id=${p.id}">View</a>`);
    });
    map.setView([center.lat, center.lng], 12);
  } catch (e) {
    results.innerHTML = `<div class="empty">${e.message}</div>`;
  }
}

form.addEventListener("submit", (e) => {
  e.preventDefault();
  loadFiltered();
});

document.getElementById("use-my-location").addEventListener("click", () => {
  if (!navigator.geolocation) {
    alert("Geolocation not supported");
    return;
  }
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      center = { lat: pos.coords.latitude, lng: pos.coords.longitude };
      map.setView([center.lat, center.lng], 13);
      loadNearby();
    },
    () => alert("Could not get your location")
  );
});

initMap();
loadFiltered();
