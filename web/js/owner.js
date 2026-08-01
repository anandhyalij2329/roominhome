renderNav("owner");

const user = getUser();
if (!getToken() || user?.role !== "owner") {
  location.href = "/login.html";
}

const formWrap = document.getElementById("form-wrap");
const ptype = document.getElementById("ptype");
const latInput = document.getElementById("lat");
const lngInput = document.getElementById("lng");
const locStatus = document.getElementById("loc-status");
const locCoords = document.getElementById("loc-coords");
let pendingFiles = [];

const placeholders = {
  room: "e.g. Private room in Baner",
  home: "e.g. 2BHK family flat in Kothrud",
  pg: "e.g. Boys PG 2 sharing near college",
  shop: "e.g. Retail shop on JM Road",
};

const amenityHints = {
  room: "wifi, ac, geyser, attached_bathroom",
  home: "lift, security, society, wifi, parking",
  pg: "wifi, meals, housekeeping, cctv, washing_machine",
  shop: "power_backup, cctv, parking, washroom",
};

document.getElementById("toggle-form").addEventListener("click", () => {
  formWrap.classList.toggle("hidden");
});

function applyTypeUI(type) {
  document.querySelectorAll(".type-fields").forEach((block) => {
    const types = (block.dataset.types || "").split(",").map((s) => s.trim());
    block.classList.toggle("hidden", !types.includes(type));
    // disable hidden inputs so they don't conflict / submit wrong required
    block.querySelectorAll("input, select, textarea").forEach((el) => {
      el.disabled = !types.includes(type);
    });
  });

  document.getElementById("title-input").placeholder = placeholders[type] || "";
  document.getElementById("amenities").placeholder = amenityHints[type] || "";

  // preferred tenants by type
  const tenantMap = {
    room: ["bachelor", "student", "family", "couple", "anyone"],
    home: ["family", "couple", "anyone"],
    pg: ["bachelor", "student", "anyone"],
    shop: ["company", "anyone"],
  };
  const allowed = new Set(tenantMap[type] || ["anyone"]);
  document.querySelectorAll("#tenants label[data-tenant]").forEach((lab) => {
    const key = lab.dataset.tenant;
    const show = allowed.has(key);
    lab.style.display = show ? "" : "none";
    const cb = lab.querySelector("input");
    if (!show) cb.checked = false;
  });

  // extras
  document.querySelectorAll("#extras [data-extra]").forEach((lab) => {
    const key = lab.dataset.extra;
    let show = false;
    if (key === "meals") show = type === "pg";
    if (key === "power" || key === "washroom") show = type === "shop";
    lab.classList.toggle("hidden", !show);
    if (!show) lab.querySelector("input").checked = false;
  });
}

ptype.addEventListener("change", () => applyTypeUI(ptype.value));
applyTypeUI(ptype.value);

function setLatLng(lat, lng) {
  latInput.value = Number(lat).toFixed(6);
  lngInput.value = Number(lng).toFixed(6);
  locCoords.textContent = `${latInput.value}, ${lngInput.value}`;
}

document.getElementById("btn-current-loc").addEventListener("click", () => {
  if (!navigator.geolocation) {
    locStatus.textContent = "Location not supported on this device.";
    return;
  }
  locStatus.textContent = "Getting current location…";
  navigator.geolocation.getCurrentPosition(
    async (pos) => {
      const lat = pos.coords.latitude;
      const lng = pos.coords.longitude;
      setLatLng(lat, lng);
      locStatus.textContent = "Current location selected.";
      // Always fill address fields from GPS (overwrite old city like Pune)
      try {
        const url = `https://nominatim.openstreetmap.org/reverse?format=jsonv2&lat=${lat}&lon=${lng}`;
        const res = await fetch(url, { headers: { Accept: "application/json" } });
        const data = await res.json();
        const a = data.address || {};
        const form = document.getElementById("property-form");
        form.city.value = a.city || a.town || a.village || a.state_district || form.city.value || "";
        form.locality.value =
          a.suburb || a.neighbourhood || a.city_district || a.county || form.locality.value || "";
        form.state.value = a.state || form.state.value || "";
        form.pincode.value = a.postcode || form.pincode.value || "";
        const road = [a.house_number, a.road].filter(Boolean).join(", ");
        if (road) form.address.value = road;
        else if (!form.address.value && data.display_name) {
          form.address.value = data.display_name.split(",").slice(0, 3).join(",").trim();
        }
        locStatus.textContent = "Current location + address filled.";
      } catch {
        // coords alone are enough
      }
    },
    () => {
      locStatus.textContent = "Location permission blocked. Allow location and try again.";
    },
    { enableHighAccuracy: true, timeout: 12000 }
  );
});

const dropzone = document.getElementById("dropzone");
const fileInput = document.getElementById("files");
const preview = document.getElementById("preview");

dropzone.addEventListener("click", () => fileInput.click());
dropzone.addEventListener("dragover", (e) => {
  e.preventDefault();
  dropzone.classList.add("drag");
});
dropzone.addEventListener("dragleave", () => dropzone.classList.remove("drag"));
dropzone.addEventListener("drop", (e) => {
  e.preventDefault();
  dropzone.classList.remove("drag");
  addFiles(e.dataTransfer.files);
});
fileInput.addEventListener("change", () => addFiles(fileInput.files));

function addFiles(fileList) {
  for (const f of fileList) pendingFiles.push(f);
  renderPreview();
}

function renderPreview() {
  preview.innerHTML = pendingFiles
    .map((f) => {
      const url = URL.createObjectURL(f);
      const tag = f.type.startsWith("video/")
        ? `<video src="${url}" muted></video>`
        : `<img src="${url}" alt="" />`;
      return `<figure title="${f.name}">${tag}</figure>`;
    })
    .join("");
}

function activeTypeBlock() {
  return document.querySelector(`.type-fields[data-types="${ptype.value}"]`);
}

function fieldInActive(name) {
  const block = activeTypeBlock();
  if (!block) return null;
  return block.querySelector(`[name="${name}"]`);
}

function val(name) {
  const el = fieldInActive(name);
  if (!el || el.disabled) return "";
  return el.value;
}

function num(name) {
  const v = val(name);
  return v === "" ? 0 : Number(v);
}

function checked(form, name) {
  const el = form.elements.namedItem(name);
  if (!el) return false;
  // extras are outside type blocks
  if (el.disabled) return false;
  return !!el.checked;
}

document.getElementById("property-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const msg = document.getElementById("form-msg");
  const form = e.target;
  const type = form.type.value;

  if (!latInput.value || !lngInput.value) {
    msg.className = "alert";
    msg.textContent = "Please click “Use current location” first.";
    return;
  }

  const tenants = [...document.querySelectorAll("#tenants label")]
    .filter((lab) => lab.style.display !== "none")
    .map((lab) => lab.querySelector("input"))
    .filter((cb) => cb && cb.checked)
    .map((cb) => cb.value);

  const amenities = String(form.amenities.value || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

  const body = {
    type,
    title: form.title.value,
    description: form.description.value,
    address: form.address.value,
    locality: form.locality.value,
    city: form.city.value,
    state: form.state.value,
    pincode: form.pincode.value,
    landmark: form.landmark.value,
    latitude: Number(latInput.value),
    longitude: Number(lngInput.value),
    rent: Number(form.rent.value),
    deposit: Number(form.deposit.value || 0),
    available_from: form.available_from.value,
    available_until: form.available_until.value,
    contact_phone: form.contact_phone.value,
    furnishing: val("furnishing"),
    sharing_type: val("sharing_type"),
    gender_preference: val("gender_preference") || "any",
    bhk: num("bhk"),
    bedrooms: num("bedrooms"),
    bathrooms: num("bathrooms"),
    area_sq_ft: num("area_sq_ft"),
    floor: num("floor"),
    total_floors: num("total_floors"),
    food_type: val("food_type"),
    shop_category: val("shop_category"),
    frontage_ft: num("frontage_ft"),
    parking_two_wheeler: checked(form, "parking_two_wheeler"),
    parking_four_wheeler: checked(form, "parking_four_wheeler"),
    meals_included: checked(form, "meals_included"),
    power_backup: checked(form, "power_backup"),
    washroom: checked(form, "washroom"),
    preferred_tenants: tenants.length ? tenants : ["anyone"],
    amenities,
  };

  if (type === "room" && !body.sharing_type) body.sharing_type = "private";
  if (type === "home" && !body.bedrooms && body.bhk) body.bedrooms = body.bhk;

  try {
    const created = await api("/api/properties", { method: "POST", body: JSON.stringify(body) });
    if (pendingFiles.length) {
      const mediaFd = new FormData();
      pendingFiles.forEach((f) => mediaFd.append("files", f));
      await api(`/api/properties/${created.id}/media`, { method: "POST", body: mediaFd });
    }
    msg.className = "alert ok";
    msg.textContent = "Property saved.";
    pendingFiles = [];
    renderPreview();
    form.reset();
    latInput.value = "";
    lngInput.value = "";
    locCoords.textContent = "";
    locStatus.textContent = "Location not set yet.";
    applyTypeUI("room");
    loadMine();
    formWrap.classList.add("hidden");
  } catch (err) {
    msg.className = "alert";
    msg.textContent = err.message;
  }
});

async function loadMine() {
  const el = document.getElementById("owner-list");
  try {
    const list = await api("/api/my/properties");
    if (!list.length) {
      el.innerHTML = `<div class="empty">No listings yet. Click “Add property”.</div>`;
      return;
    }
    el.innerHTML = list
      .map(
        (p, i) => `
      <div class="listing" style="animation-delay:${i * 0.05}s">
        <a href="/property.html?id=${p.id}" class="listing-media"><img src="${coverUrl(p)}" alt="" /></a>
        <div class="listing-body">
          <div class="listing-type">${typeLabel(p.type)} · ${p.status}</div>
          <h3><a href="/property.html?id=${p.id}">${p.title}</a></h3>
          <div class="meta">${locationLine(p)}</div>
          <div class="rent">${formatRent(p.rent)}</div>
          <div style="display:flex;gap:0.4rem;margin-top:0.5rem;flex-wrap:wrap">
            <label class="btn btn-ghost" style="padding:0.45rem 0.7rem;cursor:pointer">
              Add media
              <input type="file" multiple accept="image/*,video/*" hidden data-upload="${p.id}" />
            </label>
            <button type="button" class="btn btn-danger" data-del="${p.id}" style="padding:0.45rem 0.7rem">Delete</button>
          </div>
        </div>
      </div>`
      )
      .join("");

    el.querySelectorAll("input[data-upload]").forEach((input) => {
      input.addEventListener("change", async () => {
        const id = input.dataset.upload;
        const fd = new FormData();
        [...input.files].forEach((f) => fd.append("files", f));
        try {
          await api(`/api/properties/${id}/media`, { method: "POST", body: fd });
          loadMine();
        } catch (err) {
          alert(err.message);
        }
      });
    });
    el.querySelectorAll("button[data-del]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!confirm("Delete this property?")) return;
        try {
          await api(`/api/properties/${btn.dataset.del}`, { method: "DELETE" });
          loadMine();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  } catch (e) {
    el.innerHTML = `<div class="empty">${e.message}</div>`;
  }
}

async function loadBookings() {
  const el = document.getElementById("owner-bookings");
  try {
    const list = await api("/api/bookings");
    if (!list.length) {
      el.innerHTML = `<div class="empty">No booking requests yet.</div>`;
      return;
    }
    el.innerHTML = list
      .map(
        (b) => `
      <div class="panel" style="display:flex;flex-wrap:wrap;justify-content:space-between;gap:0.75rem;align-items:center">
        <div>
          <strong>${b.property_title || "Property"}</strong>
          <div class="muted">${b.seeker_name || "Seeker"} · ${String(b.start_date).slice(0, 10)} → ${String(b.end_date).slice(0, 10)} · ${b.status}</div>
          <div class="muted">${b.message || ""}</div>
        </div>
        <div style="display:flex;gap:0.4rem">
          ${
            b.status === "pending"
              ? `<button class="btn btn-primary" data-status="approved" data-id="${b.id}">Approve</button>
                 <button class="btn btn-ghost" data-status="rejected" data-id="${b.id}">Reject</button>`
              : ""
          }
        </div>
      </div>`
      )
      .join("");

    el.querySelectorAll("button[data-status]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/bookings/${btn.dataset.id}/status`, {
            method: "PATCH",
            body: JSON.stringify({ status: btn.dataset.status }),
          });
          loadBookings();
          loadMine();
        } catch (err) {
          alert(err.message);
        }
      });
    });
  } catch (e) {
    el.innerHTML = `<div class="empty">${e.message}</div>`;
  }
}

loadMine();
loadBookings();
