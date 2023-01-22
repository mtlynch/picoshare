import { settingsPut } from "./controllers/settings.js";
import { showElement, hideElement } from "./lib/bulma.js";

const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");

document.getElementById("settings-form").addEventListener("submit", (evt) => {
  evt.preventDefault();

  hideElement(errorContainer);
  showElement(progressSpinner);

  const defaultExpirationDays = parseInt(
    document.getElementById("default-expiration-days").value
  );

  settingsPut(defaultExpirationDays)
    .then(() => {
      // TODO
      console.log("success");
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
});
