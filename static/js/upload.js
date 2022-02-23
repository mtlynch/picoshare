import { uploadFile } from "./controllers/upload.js";

function dateInFuture(daysFromNow) {
  let d = new Date();
  d.setDate(d.getDate() + daysFromNow);
  return d;
}

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const errorContainer = document.getElementById("error");

const expirationContainer = document.querySelector(".expiration-container");
const expirationSelect = document.getElementById("expiration-select");
const expirationTimes = {
  "1 day": dateInFuture(1),
  "7 days": dateInFuture(7),
  "30 days": dateInFuture(30),
  "1 year": dateInFuture(365),
};
const defaultExpiration = "7 days";
for (const [k, v] of Object.entries(expirationTimes)) {
  const selectOption = document.createElement("option");
  selectOption.innerText = k;
  selectOption.value = v.toISOString();
  if (k === defaultExpiration) {
    selectOption.selected = true;
  }
  expirationSelect.appendChild(selectOption);
}

document
  .querySelector('.file-input[name="resume"]')
  .addEventListener("change", (evt) => {
    errorContainer.classList.add("is-hidden");
    uploadFile(evt.target.files[0], expirationSelect.value)
      .then((res) => {
        const entryId = res.id;

        const aEl = document.createElement("a");

        aEl.href = `/!${entryId}`;
        aEl.innerText = `${document.location.href}!${entryId}`;

        resultEl.appendChild(aEl);
        uploadEl.style.display = "none";
        expirationContainer.style.display = "none";
      })
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        errorContainer.classList.remove("is-hidden");
      });
  });
