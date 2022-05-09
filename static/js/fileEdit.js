// Note: I don't know of a cleaner way of doing this. datepicker doesn't seem
// to export values like a proper JS module, so we import, which populates
// window.datepicker.
import "/third-party/js-datepicker@5.18.0/js-datepicker.js";

import { editFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";

const editForm = document.getElementById("edit-form");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const expireCheckbox = document.getElementById("expire-checkbox");
const expirationInput = document.getElementById("expiration");

const picker = window.datepicker(expirationInput, {
  minDate: tomorrow(),
  dateSelected: defaultExpirationDate(),
  formatter: (input, date) => {
    input.value = date.toLocaleDateString();
  },
  respectDisabledReadOnly: true,
});

function readFilename() {
  return document.getElementById("filename").value || null;
}

function readExpiration() {
  if (!expireCheckbox.checked) {
    return null;
  }
  return picker.dateSelected.toISOString();
}

function readNote() {
  return document.getElementById("note").value || null;
}

function tomorrow() {
  return dateInNDays(1);
}

function dateInNDays(n) {
  let d = new Date();
  d.setDate(d.getDate() + n);
  return d;
}

function defaultExpirationDate() {
  const expirationRaw = expirationInput.getAttribute("data-expiration-raw");

  if (!expirationRaw) {
    return dateInNDays(30);
  }
  return new Date(expirationRaw);
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

expireCheckbox.addEventListener("change", () => {
  if (expireCheckbox.checked) {
    expirationInput.removeAttribute("disabled");
  } else {
    expirationInput.setAttribute("disabled", true);
  }
});
