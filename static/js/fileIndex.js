import { deleteFile } from "./controllers/files.js";
import { showElement, hideElement } from "./lib/bulma.js";
import { copyToClipboard } from "./lib/clipboard.js";
import { makeShortLink } from "./lib/links.js";

const errorContainer = document.getElementById("error");

document.querySelectorAll('[pico-purpose="delete"]').forEach((deleteBtn) => {
  deleteBtn.addEventListener("click", () => {
    const id = deleteBtn.getAttribute("pico-entry-id");
    deleteFile(id)
      .then(() => {
        let currentEl = deleteBtn.parentElement;
        while (currentEl && currentEl.nodeName !== "TR") {
          currentEl = currentEl.parentElement;
        }
        if (!currentEl) {
          return;
        }

        currentEl.classList.add("deleted-entry");
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
    const fileId = copyBtn.getAttribute("pico-entry-id");
    const shortLink = makeShortLink(fileId);

    copyToClipboard(shortLink)
      .then(() =>
        document
          .querySelector("snackbar-notifications")
          .addInfoMessage("Copied link")
      )
      .catch((error) => {
        document.getElementById("error-message").innerText = error;
        showElement(errorContainer);
      });
  });
});
