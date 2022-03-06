import { uploadFile } from "./controllers/upload.js";

function dateInFuture(daysFromNow) {
  let d = new Date();
  d.setDate(d.getDate() + daysFromNow);
  return d;
}

function hideElement(el) {
  el.classList.add("is-hidden");
}

function showElement(el) {
  el.classList.remove("is-hidden");
}

const uploadEl = document.querySelector(".file");
const resultEl = document.getElementById("upload-result");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const uploadForm = document.getElementById("upload-form");

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

function doUpload(file, expiration) {
  hideElement(errorContainer);
  hideElement(uploadForm);
  showElement(progressSpinner);
  uploadFile(file, expiration)
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
      showElement(errorContainer);
      showElement(uploadForm);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
}

document
  .querySelector('.file-input[name="resume"]')
  .addEventListener("change", (evt) => {
    doUpload(evt.target.files[0], expirationSelect.value);
  });

uploadForm.addEventListener("drop", (evt) => {
  evt.preventDefault();

  uploadForm.classList.remove("accepting-drop");

  if (!evt.dataTransfer.items) {
    return;
  }
  for (var i = 0; i < evt.dataTransfer.items.length; i++) {
    if (evt.dataTransfer.items[i].kind === "file") {
      var file = evt.dataTransfer.items[i].getAsFile();
      doUpload(file, expirationSelect.value);
      return;
    }
  }
});

uploadEl.addEventListener("dragover", (evt) => {
  evt.preventDefault();

  uploadEl.classList.add("accepting-drop");
});

uploadEl.addEventListener("dragenter", (evt) => {
  evt.preventDefault();

  uploadEl.classList.add("accepting-drop");
});

uploadEl.addEventListener("dragleave", (evt) => {
  uploadEl.classList.remove("accepting-drop");
});
