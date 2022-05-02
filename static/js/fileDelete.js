import { deleteFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";

const deleteForm = document.getElementById("delete-form");
const errorContainer = document.getElementById("error");
const progressSpinner = document.getElementById("progress-spinner");

document.getElementById("delete-btn").addEventListener("click", (evt) => {
  evt.preventDefault();
  const id = evt.target.getAttribute("data-entry-id");
  if (!id) {
    return;
  }
  hideElement(errorContainer);
  hideElement(deleteForm);
  showElement(progressSpinner);

  deleteFile(id)
    .then(() => {
      document.location = "/files";
    })
    .catch((error) => {
      document.getElementById("error-message").innerText = error;
      showElement(errorContainer);
      showElement(deleteForm);
    })
    .finally(() => {
      hideElement(progressSpinner);
    });
});
