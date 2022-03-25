import { deleteFile } from "./controllers/delete.js";
import { copyToClipboard } from "./lib/clipboard.js";
import { makeShortLink } from "./lib/links.js";
import { createSnackbar } from "./lib/snackbar.js";

const errorContainer = document.getElementById("error");

function hideElement(el) {
  el.classList.add("is-hidden");
}

function showElement(el) {
  el.classList.remove("is-hidden");
}

document.querySelectorAll('[pico-purpose="delete"]').forEach((deleteBtn) => {
  deleteBtn.addEventListener("click", () => {
    const id = deleteBtn.getAttribute("pico-entry-id");
    deleteFile(id)
      .then(() => {
        const rowEl = deleteBtn.parentElement.parentElement;
        rowEl.classList.add("deleted-entry");
      })
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        showElement(errorContainer);
      });
  });
});

document.querySelector("#error .delete").addEventListener("click", () => {
  hideElement(errorContainer);
});

document.querySelectorAll('[pico-purpose="copy"]').forEach((copyBtn) => {
  copyBtn.addEventListener("click", () => {
    const picoId = copyBtn.getAttribute("pico-entry-id");
    const shortLink = makeShortLink(picoId);

    copyToClipboard(shortLink)
      .then(() => createSnackbar("Link was copied!"))
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        showElement(errorContainer);
      });
  });
});
