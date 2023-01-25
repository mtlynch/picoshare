import { settingsPut } from "./controllers/settings.js";
import { showElement, hideElement } from "./lib/bulma.js";

const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const defaultExpiration = document.getElementById("default-expiration");
const timeUnit = document.getElementById("time-unit");
const saveBtn = document.querySelector("#settings-form button[type='submit']");

const daysPerYear = 365;

function readDefaultFileExpiration() {
  let defaultExpiration = parseInt(
    document.getElementById("default-expiration").value
  );
  if (timeUnit.value === "years") {
    defaultExpiration *= daysPerYear;
  }
  return defaultExpiration;
}

function disableSaveButton() {
  saveBtn.setAttribute("disabled", "true");
}

function enableSaveButton() {
  saveBtn.removeAttribute("disabled");
}

defaultExpiration.addEventListener("input", () => {
  enableSaveButton();
});

timeUnit.addEventListener("change", (evt) => {
  const maxExpirationInYears = 10;
  if (evt.target.value === "years") {
    defaultExpiration.setAttribute("max", maxExpirationInYears);
  } else {
    defaultExpiration.setAttribute("max", daysPerYear * maxExpirationInYears);
  }
  enableSaveButton();
});

document.getElementById("settings-form").addEventListener("submit", (evt) => {
  evt.preventDefault();

  hideElement(errorContainer);
  showElement(progressSpinner);
  disableSaveButton();

  settingsPut(readDefaultFileExpiration())
    .then(() => {
      document
        .querySelector("snackbar-notifications")
        .addInfoMessage("Settings saved");
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
      enableSaveButton();
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
});
