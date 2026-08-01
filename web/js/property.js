const id = qs("id");
const root = document.getElementById("detail");

if (!id) {
  root.innerHTML = `<div class="empty">Missing property id.</div>`;
} else {
  load();
}

function fullAddress(p) {
  return [p.address, p.locality, p.city, p.state, p.pincode].filter(Boolean).join(", ");
}

function phoneHref(phone) {
  const digits = String(phone || "").replace(/[^\d+]/g, "");
  return digits ? `tel:${digits}` : "";
}

function haversineKm(lat1, lon1, lat2, lon2) {
  const R = 6371;
  const dLat = ((lat2 - lat1) * Math.PI) / 180;
  const dLon = ((lon2 - lon1) * Math.PI) / 180;
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((lat1 * Math.PI) / 180) * Math.cos((lat2 * Math.PI) / 180) * Math.sin(dLon / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(a));
}

async function load() {
  try {
    const p = await api(`/api/properties/${id}`);
    document.title = `${p.title} - RoomInHome`;
    renderNav(p.type || "");
    const media = p.media || [];
    const photos = media.filter((m) => m.type === "photo");
    const videos = media.filter((m) => m.type === "video");
    const main = photos[0] || videos[0];
    const mainHtml = main
      ? main.type === "video"
        ? `<video src="${main.url}" controls></video>`
        : `<img src="${main.url}" alt="" />`
      : `<img src="${coverUrl(p)}" alt="" />`;

    const thumbs = [...photos, ...videos]
      .map(
        (m, i) => `
      <button type="button" data-i="${i}" class="${i === 0 ? "active" : ""}">
        ${m.type === "video" ? `<video src="${m.url}" muted></video>` : `<img src="${m.url}" alt="" />`}
      </button>`
      )
      .join("");

    const amenities = (p.amenities || []).map((a) => `<span class="chip">${a.replaceAll("_", " ")}</span>`).join("");
    const tenants = (p.preferred_tenants || []).map((t) => `<span class="chip">${t}</span>`).join("");
    const phone = p.contact_phone || p.owner_phone || "";
    const callLink = phoneHref(phone);
    const directions =
      p.directions_url ||
      (p.latitude || p.longitude
        ? `https://www.google.com/maps/dir/?api=1&destination=${p.latitude},${p.longitude}`
        : "");

    root.innerHTML = `
      <div class="prop-wrap">
        <div class="prop-layout">
          <div class="prop-gallery gallery">
            <div class="gallery-main" id="gallery-main">${mainHtml}</div>
            <div class="thumbs" id="thumbs">${thumbs || ""}</div>
          </div>

          <aside class="prop-side">
            <div class="prop-card">
              <div class="type-pill">${typeLabel(p.type)}</div>
              <h1>${p.title}</h1>
              <div class="prop-rent">
                <strong>${formatRent(p.rent)}</strong>
                <span>/ month</span>
              </div>
              <p class="prop-meta">Deposit ${formatRent(p.deposit)} · ${String(p.available_from).slice(0, 10)} → ${String(p.available_until).slice(0, 10)}</p>
              <div class="distance-badge" id="distance-line">Distance: detecting…</div>
            </div>

            <div class="prop-card address-block">
              <h3 class="prop-section-title">Address</h3>
              <p>${fullAddress(p)}</p>
              ${p.landmark ? `<p class="muted">Landmark: ${p.landmark}</p>` : ""}
              ${directions ? `<a class="btn btn-ghost" style="margin-top:0.55rem" href="${directions}" target="_blank" rel="noopener">Get directions</a>` : ""}
            </div>

            <div class="prop-card">
              <h3 class="prop-section-title">About</h3>
              <p class="about-text">${p.description || "No description yet."}</p>
              <div class="chip-list" style="margin-bottom:0.55rem">${tenants}</div>
              <div class="chip-list">${amenities}</div>
              <ul class="feature-list" style="margin-top:0.75rem">
                ${p.sharing_type ? `<li>Sharing: ${p.sharing_type.replaceAll("_", " ")}</li>` : ""}
                ${p.bhk ? `<li>BHK: ${p.bhk}</li>` : ""}
                ${p.bedrooms ? `<li>Bedrooms: ${p.bedrooms}</li>` : ""}
                ${p.furnishing ? `<li>Furnishing: ${p.furnishing.replaceAll("_", " ")}</li>` : ""}
                ${p.shop_category ? `<li>Shop: ${p.shop_category}</li>` : ""}
                ${p.meals_included ? `<li>Meals included (${p.food_type || "-"})</li>` : ""}
                <li>Parking: ${p.parking_two_wheeler ? "2-wheeler" : "no 2W"}${p.parking_four_wheeler ? ", 4-wheeler" : ", no 4W"}</li>
              </ul>
            </div>

            <div class="prop-card">
              <div class="owner-box">
                <div class="muted">Owner</div>
                <div class="name">${p.owner_name || "Owner"}</div>
                <div class="action-row" style="margin-top:0.75rem">
                  ${
                    phone
                      ? `<a class="btn btn-primary" href="${callLink}">Call</a>`
                      : `<span class="btn btn-ghost" style="opacity:0.6;pointer-events:none">No phone</span>`
                  }
                  ${
                    directions
                      ? `<a class="btn btn-ghost" href="${directions}" target="_blank" rel="noopener">Directions</a>`
                      : ""
                  }
                </div>
                ${phone ? `<p class="muted" style="margin:0.55rem 0 0;font-size:0.88rem">${phone}</p>` : ""}
              </div>
              <div id="book-box" style="margin-top:0.85rem"></div>
            </div>
          </aside>

          <div class="prop-map-card">
            <div class="prop-map-head">
              <h2>On the map</h2>
              <div class="prop-map-actions">
                ${directions ? `<a class="btn btn-primary" href="${directions}" target="_blank" rel="noopener">Directions</a>` : ""}
                ${p.google_map_url ? `<a class="btn btn-ghost" href="${p.google_map_url}" target="_blank" rel="noopener">Open map</a>` : ""}
              </div>
            </div>
            <div id="prop-map" class="map-box"></div>
          </div>
        </div>
      </div>`;

    const allMedia = [...photos, ...videos];
    document.getElementById("thumbs")?.addEventListener("click", (e) => {
      const btn = e.target.closest("button[data-i]");
      if (!btn) return;
      const i = Number(btn.dataset.i);
      const m = allMedia[i];
      const mainEl = document.getElementById("gallery-main");
      mainEl.innerHTML =
        m.type === "video"
          ? `<video src="${m.url}" controls autoplay></video>`
          : `<img src="${m.url}" alt="" />`;
      document.querySelectorAll("#thumbs button").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
    });

    if (p.latitude || p.longitude) {
      const map = L.map("prop-map").setView([p.latitude, p.longitude], 15);
      L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
        attribution: "&copy; OpenStreetMap",
      }).addTo(map);
      L.marker([p.latitude, p.longitude]).addTo(map);
      setTimeout(() => map.invalidateSize(), 200);
      showDistance(p, map);
    } else {
      document.getElementById("distance-line").textContent = "Distance unavailable";
    }

    renderBooking(p);
  } catch (e) {
    root.innerHTML = `<div class="empty">${e.message}</div>`;
  }
}

function showDistance(p, map) {
  const line = document.getElementById("distance-line");
  if (!navigator.geolocation) {
    line.textContent = "Enable location for distance";
    return;
  }
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      const lat = pos.coords.latitude;
      const lng = pos.coords.longitude;
      const km = Math.round(haversineKm(lat, lng, p.latitude, p.longitude) * 10) / 10;
      line.textContent = `${km} km from you`;
      L.circleMarker([lat, lng], {
        radius: 7,
        color: "#1f6b57",
        fillColor: "#1f6b57",
        fillOpacity: 0.85,
      })
        .addTo(map)
        .bindPopup("You are here");
      L.polyline(
        [
          [lat, lng],
          [p.latitude, p.longitude],
        ],
        { color: "#1f6b57", weight: 3, opacity: 0.7 }
      ).addTo(map);
      map.fitBounds(
        [
          [lat, lng],
          [p.latitude, p.longitude],
        ],
        { padding: [40, 40] }
      );
    },
    () => {
      line.textContent = "Allow location for distance";
    },
    { enableHighAccuracy: true, timeout: 10000 }
  );
}

function renderBooking(p) {
  const box = document.getElementById("book-box");
  const user = getUser();
  if (!user) {
    box.innerHTML = `<a class="btn btn-primary" style="width:100%" href="/login.html">Log in to book</a>`;
    return;
  }
  if (user.role !== "seeker") {
    box.innerHTML = `<p class="muted" style="margin:0">Switch to a seeker account to request a booking.</p>`;
    return;
  }
  box.innerHTML = `
    <form id="book-form" class="stack">
      <label>Start <input type="date" name="start_date" required /></label>
      <label>End <input type="date" name="end_date" required /></label>
      <label>Message <textarea name="message" placeholder="Tell the owner about yourself"></textarea></label>
      <button class="btn btn-primary" type="submit">Request booking</button>
      <div id="book-msg"></div>
    </form>`;
  document.getElementById("book-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const msg = document.getElementById("book-msg");
    try {
      await api("/api/bookings", {
        method: "POST",
        body: JSON.stringify({
          property_id: p.id,
          start_date: fd.get("start_date"),
          end_date: fd.get("end_date"),
          message: fd.get("message"),
        }),
      });
      msg.className = "alert ok";
      msg.textContent = "Booking request sent.";
    } catch (err) {
      msg.className = "alert";
      msg.textContent = err.message;
    }
  });
}
