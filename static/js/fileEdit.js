import { editFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";

const editForm = document.getElementById("edit-form");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const expireCheckbox = document.getElementById("expire-checkbox");
const expirationPicker = document.getElementById("expiration-picker");

function readFilename() {
  return document.getElementById("filename").value || null;
}

function readNote() {
  return document.getElementById("note").value || null;
}

document.getElementById("edit-form").addEventListener("submit", (evt) => {
  evt.preventDefault();
  const id = document.getElementById("edit-form").getAttribute("data-entry-id");
  if (!id) {
    return;
  }

  hideElement(errorContainer);
  hideElement(editForm);
  showElement(progressSpinner);

  editFile(id, readFilename(), expirationPicker.value, readNote())
    .then(() => {
      document.location = "/files";
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
      showElement(editForm);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
});

expireCheckbox.addEventListener("change", () => {
  if (expireCheckbox.checked) {
    expirationPicker.removeAttribute("disabled");
  } else {
    expirationPicker.setAttribute("disabled", true);
  }
});
