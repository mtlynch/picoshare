import { settingsPut } from "./controllers/settings.js";
import { showElement, hideElement } from "./lib/bulma.js";

const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");
const defaultExpiration = document.getElementById("default-expiration");
const timeUnit = document.getElementById("time-unit");

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

timeUnit.addEventListener("change", (evt) => {
  const maxExpirationInYears = 10;
  if (evt.target.value === "years") {
    defaultExpiration.setAttribute("max", maxExpirationInYears);
  } else {
    defaultExpiration.setAttribute("max", daysPerYear * maxExpirationInYears);
  }
});

document.getElementById("settings-form").addEventListener("submit", (evt) => {
  evt.preventDefault();

  hideElement(errorContainer);
  showElement(progressSpinner);

  settingsPut(readDefaultFileExpiration())
    .then(() => {
      // TODO
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
});
