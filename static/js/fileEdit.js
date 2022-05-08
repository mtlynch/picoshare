import datepicker from "/third-party/js-datepicker@5.18.0/js-datepicker.js";
import { editFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";

const editForm = document.getElementById("edit-form");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const picker = datepicker(document.getElementById("expiration"), {
  minDate: tomorrow(),
  formatter: (input, date) => {
    input.value = date.toLocaleDateString();
  },
});

function readFilename() {
  return document.getElementById("filename").value || null;
}

function readExpiration() {
  const expiration = picker.dateSelected;
  if (!expiration) {
    return null;
  }
  return expiration.toISOString();
}

function readNote() {
  return document.getElementById("note").value || null;
}

function tomorrow() {
  let d = new Date();
  d.setDate(d.getDate() + 1);
  return d;
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

  editFile(id, readFilename(), readExpiration(), readNote())
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
