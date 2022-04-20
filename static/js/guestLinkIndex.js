import { guestLinkDelete } from "./controllers/guestLinks.js";
import { copyToClipboard } from "./lib/clipboard.js";

const errorContainer = document.getElementById("error");

function hideElement(el) {
  el.classList.add("is-hidden");
}

function showElement(el) {
  el.classList.remove("is-hidden");
}

function makeGuestLink(linkId) {
  return `${window.location.origin}/g/${linkId}`;
}

document.querySelectorAll('[pico-purpose="delete"]').forEach((deleteBtn) => {
  deleteBtn.addEventListener("click", () => {
    const id = deleteBtn.getAttribute("pico-link-id");
    guestLinkDelete(id)
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
    const linkId = copyBtn.getAttribute("pico-link-id");
    const guestLink = makeGuestLink(linkId);

    copyToClipboard(guestLink)
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
